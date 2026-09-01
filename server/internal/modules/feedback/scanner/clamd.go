// Package scanner implements the bounded ClamD INSTREAM protocol over a local
// Unix socket. There is deliberately no network URL or scan-bypass option.
package scanner

import (
	"bytes"
	"context"
	"encoding/binary"
	"io"
	"net"
	"path/filepath"
	"strings"
	"time"

	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
)

type Clamd struct{ Socket string }

func (s Clamd) Scan(ctx context.Context, data []byte) error {
	if !filepath.IsAbs(s.Socket) || len(data) == 0 || int64(len(data)) > f.MaxImageBytes {
		return f.ErrStorage
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, "unix", s.Socket)
	if err != nil {
		return f.ErrStorage
	}
	defer conn.Close()
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	deadline, _ := ctx.Deadline()
	if conn.SetDeadline(deadline) != nil {
		return f.ErrStorage
	}
	var header [4]byte
	binary.BigEndian.PutUint32(header[:], uint32(len(data)))
	for _, part := range [][]byte{[]byte("zINSTREAM\x00"), header[:], data, {0, 0, 0, 0}} {
		if _, err = io.Copy(conn, bytes.NewReader(part)); err != nil {
			return f.ErrStorage
		}
	}
	// Non-session ClamD closes the stream after its final NUL-terminated result.
	// Accept only one exact success record, never a partial/unknown response.
	reply, err := io.ReadAll(io.LimitReader(conn, 4097))
	if err != nil || len(reply) > 4096 {
		return f.ErrStorage
	}
	if string(reply) == "stream: OK\x00" {
		return nil
	}
	if strings.HasSuffix(string(reply), " FOUND\x00") {
		return f.ErrInvalid
	}
	return f.ErrStorage
}

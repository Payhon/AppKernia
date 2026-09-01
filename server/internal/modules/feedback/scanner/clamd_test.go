package scanner

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	f "github.com/appkernia/appkernia/server/internal/modules/feedback/domain"
)

func TestClamdProtocol(t *testing.T) {
	for _, tc := range []struct {
		name, reply string
		want        error
	}{
		{"clean", "stream: OK\x00", nil},
		{"infected", "stream: Test.Signature FOUND\x00", f.ErrInvalid},
		{"engine error", "stream: scan failed ERROR\x00", f.ErrStorage},
		{"unknown", "OK\x00", f.ErrStorage},
		{"truncated", "stream: OK", f.ErrStorage},
		{"extra result", "stream: OK\x00stream: scan failed ERROR\x00", f.ErrStorage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir, err := os.MkdirTemp("/tmp", "ak-clamd-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(dir)
			socket := filepath.Join(dir, "scan.sock")
			ln, err := net.Listen("unix", socket)
			if err != nil {
				t.Fatal(err)
			}
			defer ln.Close()
			data := bytes.Repeat([]byte("image input"), 4096)
			result := make(chan error, 1)
			go func() {
				conn, e := ln.Accept()
				if e != nil {
					result <- e
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(3 * time.Second))
				reader := bufio.NewReader(conn)
				cmd, e := reader.ReadString(0)
				if e != nil || cmd != "zINSTREAM\x00" {
					result <- errors.New("incorrect command framing")
					return
				}
				var n uint32
				if e = binary.Read(reader, binary.BigEndian, &n); e != nil || n != uint32(len(data)) {
					result <- errors.New("incorrect chunk size")
					return
				}
				got := make([]byte, n)
				if _, e = io.ReadFull(reader, got); e != nil || !bytes.Equal(got, data) {
					result <- errors.New("stream bytes differ")
					return
				}
				if e = binary.Read(reader, binary.BigEndian, &n); e != nil || n != 0 {
					result <- errors.New("missing stream terminator")
					return
				}
				_, e = io.WriteString(conn, tc.reply)
				result <- e
			}()
			err = (Clamd{Socket: socket}).Scan(context.Background(), data)
			if !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if err = <-result; err != nil {
				t.Fatal(err)
			}
		})
	}
}
func TestClamdUnavailableAndCancelled(t *testing.T) {
	for _, s := range []Clamd{{}, {Socket: "relative.sock"}, {Socket: "/tmp/ak-no-such-feedback-scanner.sock"}} {
		if !errors.Is(s.Scan(context.Background(), []byte("image")), f.ErrStorage) {
			t.Fatal("unavailable scanner allowed image")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if !errors.Is((Clamd{Socket: "/tmp/ak-no-such-feedback-scanner.sock"}).Scan(ctx, []byte("image")), f.ErrStorage) {
		t.Fatal("cancelled scan allowed image")
	}
}

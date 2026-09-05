//go:build !adminembed

package adminui

import "io/fs"

func embeddedFiles() (fs.FS, bool, error) {
	return nil, false, nil
}

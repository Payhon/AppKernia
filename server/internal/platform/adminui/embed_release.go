//go:build adminembed

package adminui

import (
	"embed"
	"io/fs"
)

//go:embed dist
var embedded embed.FS

func embeddedFiles() (fs.FS, bool, error) {
	root, err := fs.Sub(embedded, "dist")
	return root, true, err
}

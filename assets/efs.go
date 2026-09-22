// Package assets embeds the HTML templates and static files (htmx) into the
// binary and exposes them as two filesystems rooted at html/ and static/.
package assets

import (
	"embed"
	"io/fs"
)

//go:embed "html" "static"
var files embed.FS

var (
	HTMLFiles   = sub(files, "html")
	StaticFiles = sub(files, "static")
)

func sub(f embed.FS, dir string) fs.FS {
	sub, err := fs.Sub(f, dir)
	if err != nil {
		panic(err)
	}
	return sub
}

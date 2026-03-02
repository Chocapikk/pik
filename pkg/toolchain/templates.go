package toolchain

import (
	"bytes"
	"embed"
	"os"
	"text/template"
)

//go:embed templates/*.tpl
var tplFS embed.FS

var tpls = template.Must(template.ParseFS(tplFS, "templates/*.tpl"))

// RenderToFile renders a template to a file.
func RenderToFile(path, tplName string, data any) error {
	var buf bytes.Buffer
	if err := tpls.ExecuteTemplate(&buf, tplName, data); err != nil {
		return err
	}
	return os.WriteFile(path, buf.Bytes(), 0644)
}

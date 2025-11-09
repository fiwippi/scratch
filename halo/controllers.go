package halo

import (
	"embed"
	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/tiff"
	_ "golang.org/x/image/webp"
	"html/template"
	_ "image/jpeg"
	_ "image/png"
)

//go:embed templates/*
var templatesFs embed.FS

var tmpl *template.Template

func init() {
	tmpl = template.Must(template.New("").ParseFS(templatesFs, "templates/*"))
}

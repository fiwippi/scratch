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
	funcs := template.FuncMap{
		"inc": func(i int) int {
			return i + 1
		},
	}
	tmpl = template.Must(template.New("").Funcs(funcs).ParseFS(templatesFs, "templates/*"))
}

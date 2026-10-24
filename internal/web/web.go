package web

import (
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"path"
)

// Alles wird in die Binärdatei eingebettet, damit der Server ohne lose Dateien läuft (TECH-STACK §2(3)).
//
//go:embed templates static
var files embed.FS

// Static liefert den Ordner static/ für den Datei-Handler unter /static/.
func Static() (fs.FS, error) {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		return nil, fmt.Errorf("static sub fs: %w", err)
	}
	return sub, nil
}

// Pages baut je Seite ein eigenes Template aus Layout, Partials und Seite.
// Jede Seite definiert "content"; getrennte Sets verhindern, dass sich Seiten gegenseitig überschreiben.
func Pages() (map[string]*template.Template, error) {
	base, err := template.ParseFS(files, "templates/layout.html", "templates/partials/*.html")
	if err != nil {
		return nil, fmt.Errorf("parse layout: %w", err)
	}

	pageFiles, err := fs.Glob(files, "templates/pages/*.html")
	if err != nil {
		return nil, fmt.Errorf("list pages: %w", err)
	}

	pages := make(map[string]*template.Template, len(pageFiles))
	for _, file := range pageFiles {
		t, err := base.Clone()
		if err != nil {
			return nil, fmt.Errorf("clone layout for %s: %w", file, err)
		}
		if _, err := t.ParseFS(files, file); err != nil {
			return nil, fmt.Errorf("parse page %s: %w", file, err)
		}
		name := path.Base(file)
		pages[name[:len(name)-len(path.Ext(name))]] = t
	}
	return pages, nil
}

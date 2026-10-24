package web

import (
	"io"
	"testing"
)

// Prüft TECH-STACK §12(2): alle Templates parsen und lassen sich mit leeren Daten ausführen.
func TestPagesParse(t *testing.T) {
	pages, err := Pages()
	if err != nil {
		t.Fatalf("Pages: %v", err)
	}
	if _, ok := pages["home"]; !ok {
		t.Fatal("page home missing")
	}
	for name, page := range pages {
		if err := page.ExecuteTemplate(io.Discard, "layout", nil); err != nil {
			t.Errorf("execute %s: %v", name, err)
		}
	}
}

func TestStaticHasStylesheets(t *testing.T) {
	static, err := Static()
	if err != nil {
		t.Fatalf("Static: %v", err)
	}
	for _, f := range []string{"css/tokens.css", "css/base.css", "css/layout.css", "css/components.css"} {
		if _, err := static.Open(f); err != nil {
			t.Errorf("open %s: %v", f, err)
		}
	}
}

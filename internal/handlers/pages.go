package handlers

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"

	"github.com/LYKOS68/lab-studio/internal/store"
)

// pageData ist alles, was Layout und Kopfleiste brauchen. Meldungen sind Codes,
// den Text dazu liefert templates/partials/messages.html.
type pageData struct {
	UserName string
	Notice   string
	Login    loginForm
	Register registerForm
	Forgot   forgotForm
	Reset    resetForm
}

type forgotForm struct {
	Email string
	Error string
}

type resetForm struct {
	Token string
	Error string
}

type loginForm struct {
	Email string
	Error string
}

type registerForm struct {
	Open   bool
	Email  string
	Name   string
	Errors map[string]string
}

// notices sind die Hinweise, die per ?notice= nach einer Weiterleitung angezeigt werden dürfen.
var notices = map[string]bool{
	"verify_sent":    true,
	"verified":       true,
	"verify_invalid": true,
	"logged_out":     true,
	"reset_sent":     true,
	"reset_invalid":  true,
	"password_reset": true,
}

func (h *Handlers) Home(w http.ResponseWriter, r *http.Request) {
	var data pageData
	name, err := h.currentUserName(r)
	if err != nil {
		// Startseite bleibt erreichbar, auch wenn die Sitzung gerade nicht prüfbar ist
		slog.Error("load current user", "err", err)
	}
	data.UserName = name

	notice := r.URL.Query().Get("notice")
	if _, ok := notices[notice]; ok {
		data.Notice = notice
	}
	h.render(w, http.StatusOK, "home", data)
}

func (h *Handlers) currentUserName(r *http.Request) (string, error) {
	c, err := r.Cookie(sessionCookie)
	if err != nil {
		return "", nil // kein Cookie heißt nur: nicht angemeldet
	}
	uid, err := h.Store.SessionUser(r.Context(), c.Value)
	if errors.Is(err, store.ErrNotFound) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return h.Store.UserName(r.Context(), uid)
}

// render schreibt erst in einen Puffer, damit bei einem Template-Fehler keine halbe Seite rausgeht.
func (h *Handlers) render(w http.ResponseWriter, status int, page string, data pageData) {
	t, ok := h.Pages[page]
	if !ok {
		slog.Error("unknown page", "page", page)
		http.Error(w, "Da ist etwas schiefgelaufen.", http.StatusInternalServerError)
		return
	}
	var buf bytes.Buffer
	if err := t.ExecuteTemplate(&buf, "layout", data); err != nil {
		slog.Error("render page", "page", page, "err", err)
		http.Error(w, "Da ist etwas schiefgelaufen.", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if _, err := buf.WriteTo(w); err != nil {
		slog.Error("write page", "page", page, "err", err)
	}
}

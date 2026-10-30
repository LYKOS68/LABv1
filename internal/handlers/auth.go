package handlers

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"mime"
	"net/http"
	"net/mail"
	"net/url"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/LYKOS68/lab-studio/internal/store"
)

const (
	sessionCookie = "session"
	sessionTTL    = 7 * 24 * time.Hour
	verifyTTL     = 24 * time.Hour
	resetTTL      = 1 * time.Hour
	maxFormBytes  = 64 << 10
)

// authInput deckt Formular und JSON ab: Browser schicken Formulare, die spätere Handy-App JSON.
type authInput struct {
	Email    string `json:"email"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Token    string `json:"token"`
}

func (h *Handlers) Register(w http.ResponseWriter, r *http.Request) {
	asJSON := isJSON(r)
	in, err := readAuthInput(w, r, asJSON)
	if err != nil {
		h.registerFailed(w, asJSON, http.StatusBadRequest, in, map[string]string{"form": "bad_request"})
		return
	}

	if errs := validateRegister(in); len(errs) > 0 {
		h.registerFailed(w, asJSON, http.StatusUnprocessableEntity, in, errs)
		return
	}

	token, tokenHash, err := newLinkToken()
	if err != nil {
		slog.Error("create verify token", "err", err)
		h.registerFailed(w, asJSON, http.StatusInternalServerError, in, map[string]string{"form": "server_error"})
		return
	}

	id, err := h.Store.CreateUser(r.Context(), in.Email, in.Name, in.Password, tokenHash, verifyTTL)
	switch {
	case errors.Is(err, store.ErrEmailTaken):
		h.registerFailed(w, asJSON, http.StatusConflict, in, map[string]string{"email": "email_taken"})
		return
	case errors.Is(err, store.ErrNameTaken):
		h.registerFailed(w, asJSON, http.StatusConflict, in, map[string]string{"name": "name_taken"})
		return
	case err != nil:
		slog.Error("create user", "err", err)
		h.registerFailed(w, asJSON, http.StatusInternalServerError, in, map[string]string{"form": "server_error"})
		return
	}

	sendLinkMail("verification", in.Email, h.BaseURL+"/verify?token="+url.QueryEscape(token))

	if asJSON {
		writeJSON(w, http.StatusCreated, map[string]any{"id": id, "status": "verify_sent"})
		return
	}
	http.Redirect(w, r, "/?notice=verify_sent", http.StatusSeeOther)
}

func (h *Handlers) Login(w http.ResponseWriter, r *http.Request) {
	asJSON := isJSON(r)
	in, err := readAuthInput(w, r, asJSON)
	if err != nil {
		h.loginFailed(w, asJSON, http.StatusBadRequest, in.Email, "bad_request")
		return
	}

	id, err := h.Store.AuthUser(r.Context(), in.Email, in.Password)
	switch {
	case errors.Is(err, store.ErrInvalidCredentials):
		h.loginFailed(w, asJSON, http.StatusUnauthorized, in.Email, "invalid_credentials")
		return
	case errors.Is(err, store.ErrNotVerified):
		h.loginFailed(w, asJSON, http.StatusForbidden, in.Email, "not_verified")
		return
	case err != nil:
		slog.Error("auth user", "err", err)
		h.loginFailed(w, asJSON, http.StatusInternalServerError, in.Email, "server_error")
		return
	}

	token, err := h.Store.CreateSession(r.Context(), id, sessionTTL)
	if err != nil {
		slog.Error("create session", "err", err)
		h.loginFailed(w, asJSON, http.StatusInternalServerError, in.Email, "server_error")
		return
	}
	h.setSessionCookie(w, r, token, int(sessionTTL.Seconds()))

	if asJSON {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true})
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *Handlers) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		if err := h.Store.DeleteSession(r.Context(), c.Value); err != nil {
			slog.Error("delete session", "err", err)
		}
	}
	h.setSessionCookie(w, r, "", -1)
	http.Redirect(w, r, "/?notice=logged_out", http.StatusSeeOther)
}

func (h *Handlers) Verify(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/?notice=verify_invalid", http.StatusSeeOther)
		return
	}
	err := h.Store.VerifyEmail(r.Context(), hashToken(token))
	if errors.Is(err, store.ErrNotFound) {
		http.Redirect(w, r, "/?notice=verify_invalid", http.StatusSeeOther)
		return
	}
	if err != nil {
		slog.Error("verify email", "err", err)
		http.Redirect(w, r, "/?notice=verify_invalid", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/?notice=verified", http.StatusSeeOther)
}

// ResendVerification schickt einen neuen Bestätigungslink. Die Antwort ist immer dieselbe,
// damit sich darüber nicht herausfinden lässt, welche Adressen registriert sind.
func (h *Handlers) ResendVerification(w http.ResponseWriter, r *http.Request) {
	asJSON := isJSON(r)
	in, err := readAuthInput(w, r, asJSON)
	if err == nil && validEmail(in.Email) {
		h.sendNewVerification(r, in.Email)
	}
	if asJSON {
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "verify_sent"})
		return
	}
	http.Redirect(w, r, "/?notice=verify_sent", http.StatusSeeOther)
}

func (h *Handlers) sendNewVerification(r *http.Request, email string) {
	token, tokenHash, err := newLinkToken()
	if err != nil {
		slog.Error("create verify token", "err", err)
		return
	}
	err = h.Store.RenewVerification(r.Context(), email, tokenHash, verifyTTL)
	if errors.Is(err, store.ErrNotFound) {
		return // kein Konto oder schon bestätigt
	}
	if err != nil {
		slog.Error("renew verification", "err", err)
		return
	}
	sendLinkMail("verification", email, h.BaseURL+"/verify?token="+url.QueryEscape(token))
}

func (h *Handlers) ForgotPasswordPage(w http.ResponseWriter, r *http.Request) {
	h.render(w, http.StatusOK, "forgot-password", pageData{})
}

// ForgotPassword antwortet wie ResendVerification immer gleich, ob es das Konto gibt oder nicht.
func (h *Handlers) ForgotPassword(w http.ResponseWriter, r *http.Request) {
	asJSON := isJSON(r)
	in, err := readAuthInput(w, r, asJSON)
	if err != nil {
		h.forgotFailed(w, asJSON, http.StatusBadRequest, in.Email, "bad_request")
		return
	}
	if !validEmail(in.Email) {
		h.forgotFailed(w, asJSON, http.StatusUnprocessableEntity, in.Email, "email_invalid")
		return
	}

	token, tokenHash, err := newLinkToken()
	if err != nil {
		slog.Error("create reset token", "err", err)
		h.forgotFailed(w, asJSON, http.StatusInternalServerError, in.Email, "server_error")
		return
	}
	err = h.Store.CreatePasswordReset(r.Context(), in.Email, tokenHash, resetTTL)
	switch {
	case errors.Is(err, store.ErrNotFound):
		// kein Konto: nichts verschicken, aber genauso antworten
	case err != nil:
		slog.Error("create password reset", "err", err)
		h.forgotFailed(w, asJSON, http.StatusInternalServerError, in.Email, "server_error")
		return
	default:
		sendLinkMail("password reset", in.Email, h.BaseURL+"/reset-password?token="+url.QueryEscape(token))
	}

	if asJSON {
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "reset_sent"})
		return
	}
	http.Redirect(w, r, "/?notice=reset_sent", http.StatusSeeOther)
}

// ResetPasswordPage zeigt nur das Formular. Der Token wird erst beim Absenden eingelöst,
// weil Mail-Programme Links oft vorab öffnen, um sie zu prüfen.
func (h *Handlers) ResetPasswordPage(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Redirect(w, r, "/?notice=reset_invalid", http.StatusSeeOther)
		return
	}
	// Der Token steht in der Adresse und darf nicht per Referer an andere Seiten gehen
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.render(w, http.StatusOK, "reset-password", pageData{Reset: resetForm{Token: token}})
}

func (h *Handlers) ResetPassword(w http.ResponseWriter, r *http.Request) {
	asJSON := isJSON(r)
	in, err := readAuthInput(w, r, asJSON)
	if err != nil {
		h.resetFailed(w, asJSON, http.StatusBadRequest, in.Token, "bad_request")
		return
	}
	if code := validatePassword(in.Password); code != "" {
		h.resetFailed(w, asJSON, http.StatusUnprocessableEntity, in.Token, code)
		return
	}

	err = h.Store.ResetPassword(r.Context(), hashToken(in.Token), in.Password)
	if errors.Is(err, store.ErrNotFound) {
		h.resetFailed(w, asJSON, http.StatusBadRequest, "", "reset_invalid")
		return
	}
	if err != nil {
		slog.Error("reset password", "err", err)
		h.resetFailed(w, asJSON, http.StatusInternalServerError, in.Token, "server_error")
		return
	}

	// Alle Sitzungen sind beendet; das eigene Cookie wird mit aufgeräumt
	h.setSessionCookie(w, r, "", -1)
	if asJSON {
		writeJSON(w, http.StatusOK, map[string]any{"status": "password_reset"})
		return
	}
	http.Redirect(w, r, "/?notice=password_reset", http.StatusSeeOther)
}

func (h *Handlers) forgotFailed(w http.ResponseWriter, asJSON bool, status int, email, code string) {
	if asJSON {
		writeJSON(w, status, map[string]any{"error": code})
		return
	}
	h.render(w, status, "forgot-password", pageData{Forgot: forgotForm{Email: email, Error: code}})
}

func (h *Handlers) resetFailed(w http.ResponseWriter, asJSON bool, status int, token, code string) {
	if asJSON {
		writeJSON(w, status, map[string]any{"error": code})
		return
	}
	w.Header().Set("Referrer-Policy", "no-referrer")
	h.render(w, status, "reset-password", pageData{Reset: resetForm{Token: token, Error: code}})
}

func (h *Handlers) registerFailed(w http.ResponseWriter, asJSON bool, status int, in authInput, errs map[string]string) {
	if asJSON {
		writeJSON(w, status, map[string]any{"errors": errs})
		return
	}
	h.render(w, status, "home", pageData{
		Register: registerForm{Open: true, Email: in.Email, Name: in.Name, Errors: errs},
	})
}

func (h *Handlers) loginFailed(w http.ResponseWriter, asJSON bool, status int, email, code string) {
	if asJSON {
		writeJSON(w, status, map[string]any{"error": code})
		return
	}
	h.render(w, status, "home", pageData{Login: loginForm{Email: email, Error: code}})
}

func readAuthInput(w http.ResponseWriter, r *http.Request, asJSON bool) (authInput, error) {
	r.Body = http.MaxBytesReader(w, r.Body, maxFormBytes)
	var in authInput
	if asJSON {
		if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
			return in, fmt.Errorf("decode json: %w", err)
		}
	} else {
		if err := r.ParseForm(); err != nil {
			return in, fmt.Errorf("parse form: %w", err)
		}
		in = authInput{Email: r.PostForm.Get("email"), Name: r.PostForm.Get("name"), Password: r.PostForm.Get("password"), Token: r.PostForm.Get("token")}
	}
	// E-Mail klein und ohne Ränder speichern, damit dieselbe Adresse nicht zweimal als Konto auftaucht
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Name = strings.TrimSpace(in.Name)
	return in, nil
}

func validateRegister(in authInput) map[string]string {
	errs := map[string]string{}

	if !validEmail(in.Email) {
		errs["email"] = "email_invalid"
	}

	if n := utf8.RuneCountInString(in.Name); n < 2 || n > 30 || !validNameRunes(in.Name) {
		errs["name"] = "name_invalid"
	}

	if code := validatePassword(in.Password); code != "" {
		errs["password"] = code
	}
	return errs
}

func validEmail(email string) bool {
	addr, err := mail.ParseAddress(email)
	return len(email) <= 254 && err == nil && addr.Address == email
}

// validatePassword gibt den Fehlercode zurück oder "", wenn das Passwort passt.
func validatePassword(password string) string {
	// bcrypt liest höchstens 72 Byte; längere Passwörter würden still abgeschnitten
	switch {
	case utf8.RuneCountInString(password) < 8:
		return "password_short"
	case len(password) > 72:
		return "password_long"
	}
	return ""
}

func validNameRunes(name string) bool {
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && !strings.ContainsRune(" .-_", c) {
			return false
		}
	}
	return true
}

// newLinkToken gibt den Token für einen Mail-Link und seinen Hash für die Datenbank zurück.
// In der Datenbank liegt nur der Hash, damit ein Datenbank-Leck keine gültigen Links verrät.
func newLinkToken() (string, string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", fmt.Errorf("read random: %w", err)
	}
	token := hex.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// sendLinkMail gibt die Mail lokal in der Konsole aus (TECH-STACK §1(10)).
// SMTP kommt dazu, sobald der Anbieter für den Online-Betrieb feststeht.
func sendLinkMail(kind, to, link string) {
	slog.Info(kind+" mail (console)", "to", to, "link", link)
}

func isJSON(r *http.Request) bool {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	return err == nil && mediaType == "application/json"
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		slog.Error("write json", "err", err)
	}
}


func (h *Handlers) setSessionCookie(w http.ResponseWriter, r *http.Request, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie,
		Value: token,
		Path: "/",
		MaxAge: maxAge,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure: h.sessionCookieSecure(r),
	})
}

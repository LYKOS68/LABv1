package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/LYKOS68/lab-studio/internal/store"
)

type Handlers struct {
	Store   *store.Store
	Pages   map[string]*template.Template
	BaseURL string // Grundadresse für Links in E-Mails, z. B. http://localhost:8080
}

func (h *Handlers) Profile(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "bad id", http.StatusBadRequest)
		return
	}
	name, bio, err := h.Store.GetProfile(r.Context(), id)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":%d,"display_name":%q,"bio":%q}`, id, name, bio)
}

func (h *Handlers) Offerings(w http.ResponseWriter, r *http.Request) {
	list, err := h.Store.ListOfferings(r.Context())
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(list)
}

func (h *Handlers) Me(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	uid, err := h.Store.SessionUser(r.Context(), c.Value)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"user_id":%d}`, uid)
}

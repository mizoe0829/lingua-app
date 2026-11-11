package handler

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"lingua-backend/internal/model"
)

type LanguageHandler struct {
	db *sql.DB
}

func NewLanguageHandler(db *sql.DB) *LanguageHandler {
	return &LanguageHandler{db: db}
}

// checkAdminAuth verifies the X-Admin-Secret header or Bearer token
func checkAdminAuth(r *http.Request) bool {
	expectedSecret := os.Getenv("ADMIN_SECRET")
	if expectedSecret == "" {
		expectedSecret = "lingua-admin-2026"
	}

	// 1. Check X-Admin-Secret header
	if r.Header.Get("X-Admin-Secret") == expectedSecret {
		return true
	}

	// 2. Check Authorization: Bearer <secret>
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		if token == expectedSecret {
			return true
		}
	}

	return false
}

// GetLanguages returns all supported languages
func (h *LanguageHandler) GetLanguages(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.Query(`
		SELECT code, name, native_name, flag, voice_name, pitch, rate
		FROM languages ORDER BY code ASC`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var languages []model.Language
	for rows.Next() {
		var l model.Language
		if err := rows.Scan(&l.Code, &l.Name, &l.NativeName, &l.Flag, &l.VoiceName, &l.Pitch, &l.Rate); err == nil {
			languages = append(languages, l)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(languages)
}

// HandlePresets handles GET (public list) and POST (admin create)
func (h *LanguageHandler) HandlePresets(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.GetPresets(w, r)
	case http.MethodPost:
		h.CreatePreset(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// GetPresets returns sample learning texts, optionally filtered by ?lang=
func (h *LanguageHandler) GetPresets(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")

	query := `SELECT id, lang_code, category, title, description, content, difficulty FROM presets`
	args := []interface{}{}

	if lang != "" {
		query += ` WHERE lang_code = ?`
		args = append(args, lang)
	}

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var presets []model.PresetText
	for rows.Next() {
		var p model.PresetText
		if err := rows.Scan(&p.ID, &p.LangCode, &p.Category, &p.Title, &p.Description, &p.Content, &p.Difficulty); err == nil {
			presets = append(presets, p)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(presets)
}

// CreatePreset handles POST /api/presets (Admin only)
func (h *LanguageHandler) CreatePreset(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r) {
		http.Error(w, "Unauthorized: Invalid or missing admin credentials", http.StatusUnauthorized)
		return
	}

	var p model.PresetText
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Basic validation
	if strings.TrimSpace(p.LangCode) == "" || strings.TrimSpace(p.Title) == "" || strings.TrimSpace(p.Content) == "" {
		http.Error(w, "lang_code, title, and content are required", http.StatusBadRequest)
		return
	}

	if p.ID == "" {
		// Generate an ID like en-custom-1715420000
		prefix := strings.Split(p.LangCode, "-")[0]
		p.ID = fmt.Sprintf("%s-custom-%d", prefix, time.Now().UnixNano()%1000000)
	}

	if p.Category == "" {
		p.Category = "General"
	}
	if p.Difficulty == "" {
		p.Difficulty = "Intermediate"
	}

	_, err := h.db.Exec(`
		INSERT INTO presets (id, lang_code, category, title, description, content, difficulty)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		p.ID, p.LangCode, p.Category, p.Title, p.Description, p.Content, p.Difficulty)
	if err != nil {
		http.Error(w, "Failed to insert preset: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(p)
}

// DeletePreset handles DELETE /api/presets/{id} (Admin only)
func (h *LanguageHandler) DeletePreset(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if !checkAdminAuth(r) {
		http.Error(w, "Unauthorized: Invalid or missing admin credentials", http.StatusUnauthorized)
		return
	}

	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Missing preset ID in path", http.StatusBadRequest)
		return
	}
	id := parts[len(parts)-1]

	result, err := h.db.Exec(`DELETE FROM presets WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete preset: "+err.Error(), http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Preset not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}

// VerifyAdmin handles GET /api/admin/verify to validate secret key from frontend
func (h *LanguageHandler) VerifyAdmin(w http.ResponseWriter, r *http.Request) {
	if !checkAdminAuth(r) {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"valid": true})
}

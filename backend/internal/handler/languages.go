package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"lingua-backend/internal/model"
)

type LanguageHandler struct {
	db *sql.DB
}

func NewLanguageHandler(db *sql.DB) *LanguageHandler {
	return &LanguageHandler{db: db}
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


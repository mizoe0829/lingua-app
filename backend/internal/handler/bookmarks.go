package handler

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"lingua-backend/internal/model"
)

type BookmarkHandler struct {
	db *sql.DB
}

func NewBookmarkHandler(db *sql.DB) *BookmarkHandler {
	return &BookmarkHandler{db: db}
}

// HandleBookmarks handles GET (list) and POST (create)
func (h *BookmarkHandler) HandleBookmarks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *BookmarkHandler) list(w http.ResponseWriter, r *http.Request) {
	lang := r.URL.Query().Get("lang")
	query := `SELECT id, lang_code, text, ipa, meaning, context, notes, created_at FROM bookmarks`
	args := []interface{}{}

	if lang != "" {
		query += ` WHERE lang_code = ?`
		args = append(args, lang)
	}
	query += ` ORDER BY created_at DESC`

	rows, err := h.db.Query(query, args...)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var bookmarks []model.Bookmark
	for rows.Next() {
		var b model.Bookmark
		if err := rows.Scan(&b.ID, &b.LangCode, &b.Text, &b.IPA, &b.Meaning, &b.Context, &b.Notes, &b.CreatedAt); err == nil {
			bookmarks = append(bookmarks, b)
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(bookmarks)
}

func (h *BookmarkHandler) create(w http.ResponseWriter, r *http.Request) {
	var b model.Bookmark
	if err := json.NewDecoder(r.Body).Decode(&b); err != nil {
		http.Error(w, "Invalid JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(b.Text) == "" {
		http.Error(w, "Text cannot be empty", http.StatusBadRequest)
		return
	}

	result, err := h.db.Exec(`
		INSERT INTO bookmarks (lang_code, text, ipa, meaning, context, notes)
		VALUES (?, ?, ?, ?, ?, ?)`,
		b.LangCode, b.Text, b.IPA, b.Meaning, b.Context, b.Notes)
	if err != nil {
		http.Error(w, "Failed to insert bookmark: "+err.Error(), http.StatusInternalServerError)
		return
	}

	id, _ := result.LastInsertId()
	b.ID = id

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(b)
}

// DeleteBookmark handles DELETE /api/bookmarks/:id
func (h *BookmarkHandler) DeleteBookmark(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract ID from path suffix (e.g. /api/bookmarks/3)
	parts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	if len(parts) < 3 {
		http.Error(w, "Missing ID in path", http.StatusBadRequest)
		return
	}

	id, err := strconv.ParseInt(parts[len(parts)-1], 10, 64)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	_, err = h.db.Exec(`DELETE FROM bookmarks WHERE id = ?`, id)
	if err != nil {
		http.Error(w, "Failed to delete bookmark: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}


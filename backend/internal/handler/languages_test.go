package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"lingua-backend/internal/database"
	"lingua-backend/internal/model"
)

func TestPresetAdminCRUD(t *testing.T) {
	db, err := database.InitDB(":memory:")
	if err != nil {
		t.Fatalf("failed to init in-memory db: %v", err)
	}
	defer db.Close()

	h := NewLanguageHandler(db)

	t.Run("Public GET /api/presets works without auth", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/presets?lang=en-US", nil)
		rec := httptest.NewRecorder()
		h.GetPresets(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d", rec.Code)
		}
	})

	t.Run("POST /api/presets without admin secret returns 401 Unauthorized", func(t *testing.T) {
		body := `{"lang_code":"en-US","title":"Test","content":"Hello"}`
		req := httptest.NewRequest(http.MethodPost, "/api/presets", bytes.NewBufferString(body))
		rec := httptest.NewRecorder()
		h.CreatePreset(rec, req)

		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", rec.Code)
		}
	})

	t.Run("POST /api/presets with valid X-Admin-Secret creates preset", func(t *testing.T) {
		payload := model.PresetText{
			ID:          "en-custom-test-1",
			LangCode:    "en-US",
			Category:    "Testing",
			Title:       "Admin Custom Sentence",
			Description: "Created via automated test",
			Content:     "This is an administrative custom preset sentence.",
			Difficulty:  "Intermediate",
		}
		jsonBytes, _ := json.Marshal(payload)
		req := httptest.NewRequest(http.MethodPost, "/api/presets", bytes.NewBuffer(jsonBytes))
		req.Header.Set("X-Admin-Secret", "lingua-admin-2026")
		rec := httptest.NewRecorder()
		h.CreatePreset(rec, req)

		if rec.Code != http.StatusCreated {
			t.Fatalf("expected 201 Created, got %d: %s", rec.Code, rec.Body.String())
		}

		var created model.PresetText
		if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
			t.Fatalf("failed to parse created preset response: %v", err)
		}
		if created.ID != "en-custom-test-1" {
			t.Errorf("expected ID en-custom-test-1, got %s", created.ID)
		}
	})

	t.Run("DELETE /api/presets/:id requires admin and deletes", func(t *testing.T) {
		// Unauthorized
		reqNoAuth := httptest.NewRequest(http.MethodDelete, "/api/presets/en-custom-test-1", nil)
		recNoAuth := httptest.NewRecorder()
		h.DeletePreset(recNoAuth, reqNoAuth)
		if recNoAuth.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 Unauthorized, got %d", recNoAuth.Code)
		}

		// Authorized
		reqAuth := httptest.NewRequest(http.MethodDelete, "/api/presets/en-custom-test-1", nil)
		reqAuth.Header.Set("X-Admin-Secret", "lingua-admin-2026")
		recAuth := httptest.NewRecorder()
		h.DeletePreset(recAuth, reqAuth)
		if recAuth.Code != http.StatusOK {
			t.Errorf("expected 200 OK, got %d: %s", recAuth.Code, recAuth.Body.String())
		}
	})
}


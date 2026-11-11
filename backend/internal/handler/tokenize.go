package handler

import (
	"encoding/json"
	"net/http"

	"lingua-backend/internal/model"
	"lingua-backend/internal/service"
)

type TokenizeHandler struct {
	service *service.TokenizerService
}

func NewTokenizeHandler(service *service.TokenizerService) *TokenizeHandler {
	return &TokenizeHandler{service: service}
}

// Tokenize handles text parsing and IPA/meaning resolution
func (h *TokenizeHandler) Tokenize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.TokenizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.LangCode == "" {
		req.LangCode = "en-US"
	}

	res, err := h.service.Tokenize(req.LangCode, req.Text)
	if err != nil {
		http.Error(w, "Tokenization error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(res)
}


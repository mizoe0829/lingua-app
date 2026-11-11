package handler

import (
	"encoding/json"
	"net/http"

	"lingua-backend/internal/model"
	"lingua-backend/internal/service"
)

type AnalyzeHandler struct {
	gemini *service.GeminiService
}

func NewAnalyzeHandler(gemini *service.GeminiService) *AnalyzeHandler {
	return &AnalyzeHandler{gemini: gemini}
}

// AnalyzeStream provides real-time SSE streaming for phonetics and syntax
func (h *AnalyzeHandler) AnalyzeStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.AnalyzeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if req.Sentence == "" {
		http.Error(w, "Sentence cannot be empty", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported by server", http.StatusInternalServerError)
		return
	}

	// Set SSE Headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Call Gemini stream service
	if err := h.gemini.StreamPhoneticAnalysis(r.Context(), req, flusher, w); err != nil {
		payload, _ := json.Marshal(map[string]string{"error": err.Error()})
		w.Write([]byte("data: " + string(payload) + "\n\n"))
		flusher.Flush()
	}
}


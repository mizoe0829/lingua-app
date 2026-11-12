package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"lingua-backend/internal/model"
	"lingua-backend/internal/service"
)

type DialogHandler struct {
	gemini *service.GeminiService
}

func NewDialogHandler(gemini *service.GeminiService) *DialogHandler {
	return &DialogHandler{gemini: gemini}
}

// DialogStream handles conversational sparring streaming with level control
func (h *DialogHandler) DialogStream(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req model.DialogStreamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON payload: "+err.Error(), http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.UserMessage) == "" {
		http.Error(w, "user_message cannot be empty", http.StatusBadRequest)
		return
	}

	if req.LangCode == "" {
		req.LangCode = "en-US"
	}
	if req.Level == "" {
		req.Level = "Intermediate"
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

	if err := h.gemini.StreamDialog(r.Context(), req, flusher, w); err != nil {
		errPayload, _ := json.Marshal(model.DialogStreamChunk{
			Type:  "error",
			Chunk: err.Error(),
		})
		w.Write([]byte("data: " + string(errPayload) + "\n\n"))
		flusher.Flush()
	}
}


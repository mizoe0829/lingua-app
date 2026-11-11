package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"lingua-backend/internal/database"
	"lingua-backend/internal/handler"
	"lingua-backend/internal/service"
)

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8085"
	}

	// 1. Initialize SQLite Database
	db, err := database.InitDB("lingua.db")
	if err != nil {
		log.Fatalf("[Server] Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 2. Initialize Services
	tokenizerService := service.NewTokenizerService(db)
	geminiService := service.NewGeminiService()

	// 3. Initialize Handlers
	langHandler := handler.NewLanguageHandler(db)
	tokenHandler := handler.NewTokenizeHandler(tokenizerService)
	analyzeHandler := handler.NewAnalyzeHandler(geminiService)
	bookmarkHandler := handler.NewBookmarkHandler(db)

	// 4. Router Setup
	mux := http.NewServeMux()

	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok","app":"LinguaLight API","version":"1.0.0"}`))
	})

	mux.HandleFunc("/api/languages", langHandler.GetLanguages)
	mux.HandleFunc("/api/presets", langHandler.GetPresets)
	mux.HandleFunc("/api/tokenize", tokenHandler.Tokenize)
	mux.HandleFunc("/api/analyze/stream", analyzeHandler.AnalyzeStream)

	// Bookmark routing
	mux.HandleFunc("/api/bookmarks", bookmarkHandler.HandleBookmarks)
	mux.HandleFunc("/api/bookmarks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			bookmarkHandler.DeleteBookmark(w, r)
			return
		}
		if strings.TrimPrefix(r.URL.Path, "/api/bookmarks/") == "" {
			bookmarkHandler.HandleBookmarks(w, r)
			return
		}
		http.NotFound(w, r)
	})

	server := &http.Server{
		Addr:         ":" + port,
		Handler:      corsMiddleware(mux),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 120 * time.Second, // Allow extended SSE streams
		IdleTimeout:  60 * time.Second,
	}

	// 5. Graceful shutdown
	go func() {
		log.Printf("[LinguaLight] Server running on http://localhost:%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("[LinguaLight] Server listen error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("[LinguaLight] Gracefully shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("[LinguaLight] Server forced shutdown: %v", err)
	}
	log.Println("[LinguaLight] Server exited safely.")
}


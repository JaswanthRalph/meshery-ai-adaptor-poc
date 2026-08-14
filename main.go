package main

import (
	"embed"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/meshery/ai-adapter-poc/internal/ai"
	"github.com/meshery/ai-adapter-poc/internal/handlers"
	"github.com/meshery/ai-adapter-poc/internal/store"
)

//go:embed ui/*
var uiFiles embed.FS

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "9082"
	}

	// Initialize the store (in-memory for PoC; Meshery uses encrypted DB)
	dataStore := store.New()

	// Initialize the provider registry with all built-in providers
	registry := ai.NewRegistry()

	// Initialize the generation pipeline
	pipeline := ai.NewPipeline(registry)

	// Create and register HTTP handlers
	handler := handlers.NewHandler(dataStore, registry, pipeline)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	// Serve the embedded UI
	uiFS, err := fs.Sub(uiFiles, "ui")
	if err != nil {
		log.Fatalf("Failed to create UI filesystem: %v", err)
	}
	mux.Handle("/", http.FileServer(http.FS(uiFS)))

	// Start server
	addr := fmt.Sprintf(":%s", port)
	log.Printf("╔══════════════════════════════════════════════════════════════╗")
	log.Printf("║  Meshery AI Adapter PoC — BYOM: Bring Your Own Model       ║")
	log.Printf("║  CNCF LFX Mentorship 2026 Term 3                           ║")
	log.Printf("╠══════════════════════════════════════════════════════════════╣")
	log.Printf("║  Server:  http://localhost%s                            ║", addr)
	log.Printf("║  API:     http://localhost%s/api/ai/providers           ║", addr)
	log.Printf("║  UI:      http://localhost%s                            ║", addr)
	log.Printf("╠══════════════════════════════════════════════════════════════╣")
	log.Printf("║  Providers: OpenAI, Anthropic, Ollama, Azure OpenAI        ║")
	log.Printf("║  Secrets are encrypted at rest and never returned to clients║")
	log.Printf("╚══════════════════════════════════════════════════════════════╝")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// Copyright 2026 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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

	// Initialize the store (GORM SQLite for data persistence)
	dataStore, err := store.New()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

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

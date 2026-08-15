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

// Package handlers implements the HTTP API layer for the Meshery
// AI Adapter PoC. Routes follow the Meshery API pattern:
//
//	POST   /api/ai/connections          — Create an AI provider connection
//	GET    /api/ai/connections          — List AI connections
//	GET    /api/ai/connections/{id}     — Get a connection
//	DELETE /api/ai/connections/{id}     — Delete a connection
//	POST   /api/ai/credentials         — Create a credential
//	GET    /api/ai/credentials         — List credentials (secrets masked)
//	DELETE /api/ai/credentials/{id}    — Delete a credential
//	GET    /api/ai/connections/{id}/health — Health check
//	POST   /api/ai/generate            — NL→Design generation
//	GET    /api/ai/providers           — List supported provider kinds
package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meshery/ai-adapter-poc/internal/ai"
	"github.com/meshery/ai-adapter-poc/internal/models"
	"github.com/meshery/ai-adapter-poc/internal/store"
)

// Handler encapsulates all dependencies for the API handlers.
type Handler struct {
	store    *store.Store
	registry *ai.Registry
	pipeline *ai.Pipeline
}

// NewHandler creates a Handler with the given dependencies.
func NewHandler(s *store.Store, r *ai.Registry, p *ai.Pipeline) *Handler {
	return &Handler{store: s, registry: r, pipeline: p}
}

// RegisterRoutes sets up all API routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/ai/providers", h.corsMiddleware(h.handleProviders))
	mux.HandleFunc("/api/ai/connections", h.corsMiddleware(h.handleConnections))
	mux.HandleFunc("/api/ai/credentials", h.corsMiddleware(h.handleCredentials))
	mux.HandleFunc("/api/ai/generate", h.corsMiddleware(h.handleGenerate))
	mux.HandleFunc("/api/kanvas-export", h.corsMiddleware(h.handleKanvasExport))
	// Pattern-based routes (Go 1.22+)
	mux.HandleFunc("/api/ai/connections/", h.corsMiddleware(h.handleConnectionByID))
	mux.HandleFunc("/api/ai/credentials/", h.corsMiddleware(h.handleCredentialByID))
}

func (h *Handler) corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "http://localhost:9081" || origin == "http://localhost:9082" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Operation-ID, X-User-ID")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next(w, r)
	}
}

func (h *Handler) handleProviders(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	providers := []map[string]interface{}{
		{
			"kind": "openai", "name": "OpenAI",
			"description":    "OpenAI GPT models (GPT-4o, GPT-4, etc.)",
			"requires_creds": true,
			"config_fields":  []string{"base_url", "model", "organization_id"},
			"cred_fields":    []string{"api_key"},
			"default_config": map[string]string{"base_url": "https://api.openai.com/v1", "model": "gpt-4o"},
		},
		{
			"kind": "anthropic", "name": "Anthropic Claude",
			"description":    "Anthropic Claude models (Claude Sonnet, Opus, Haiku)",
			"requires_creds": true,
			"config_fields":  []string{"base_url", "model", "anthropic_version"},
			"cred_fields":    []string{"api_key"},
			"default_config": map[string]string{"base_url": "https://api.anthropic.com", "model": "claude-sonnet-4-20250514"},
		},
		{
			"kind": "ollama", "name": "Ollama (Local)",
			"description":    "Local inference with Ollama — no data leaves your machine",
			"requires_creds": false,
			"config_fields":  []string{"base_url", "model"},
			"cred_fields":    []string{},
			"default_config": map[string]string{"base_url": "http://localhost:11434", "model": "llama3.1"},
		},
		{
			"kind": "azure-openai", "name": "Azure OpenAI",
			"description":    "Azure-hosted OpenAI models with enterprise compliance",
			"requires_creds": true,
			"config_fields":  []string{"resource_name", "deployment_id", "api_version"},
			"cred_fields":    []string{"api_key"},
			"default_config": map[string]string{"api_version": "2024-02-15-preview"},
		},
		{
			"kind": "vertex-ai", "name": "Google Vertex AI",
			"description":    "Google Cloud Vertex AI (Gemini models)",
			"requires_creds": true,
			"config_fields":  []string{"project_id", "location", "model"},
			"cred_fields":    []string{"access_token"},
			"default_config": map[string]string{"location": "us-central1", "model": "gemini-1.5-pro"},
		},
	}
	writeJSON(w, http.StatusOK, providers)
}

func (h *Handler) handleConnections(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		conns := h.store.ListConnections("")
		writeJSON(w, http.StatusOK, conns)
	case "POST":
		var req struct {
			Name         string              `json:"name"`
			Kind         models.ProviderKind `json:"kind"`
			Config       map[string]string   `json:"config"`
			CredentialID string              `json:"credential_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: %v", err)
			return
		}
		if req.Name == "" || req.Kind == "" {
			writeError(w, http.StatusBadRequest, "name and kind are required")
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "default-user"
		}

		conn := &models.Connection{
			Name:         req.Name,
			Kind:         req.Kind,
			Config:       req.Config,
			CredentialID: req.CredentialID,
			UserID:       userID,
			Status:       models.StatusRegistered,
		}

		created, err := h.store.CreateConnection(conn)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create connection: %v", err)
			return
		}

		log.Printf("[CONNECTION] Created %s connection '%s' (id=%s) — operationId=%s",
			conn.Kind, conn.Name, created.ID, r.Header.Get("X-Operation-ID"))

		writeJSON(w, http.StatusCreated, created)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleConnectionByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/ai/connections/"), "/")
	id := parts[0]

	// Check for /health sub-route
	if len(parts) > 1 && parts[1] == "health" {
		h.handleHealthCheck(w, r, id)
		return
	}

	switch r.Method {
	case "GET":
		conn, err := h.store.GetConnection(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "Connection not found: %v", err)
			return
		}
		writeJSON(w, http.StatusOK, conn)
	case "DELETE":
		if err := h.store.DeleteConnection(id); err != nil {
			writeError(w, http.StatusNotFound, "Connection not found: %v", err)
			return
		}
		log.Printf("[CONNECTION] Deleted connection id=%s", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCredentials(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		creds := h.store.ListCredentials("")
		// SECURITY: Never return secret values — only return safe projections
		safe := make([]models.CredentialResponse, len(creds))
		for i, c := range creds {
			safe[i] = c.ToResponse()
		}
		writeJSON(w, http.StatusOK, safe)
	case "POST":
		var req struct {
			Name   string              `json:"name"`
			Kind   models.ProviderKind `json:"kind"`
			Secret map[string]string   `json:"secret"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "Invalid request body: %v", err)
			return
		}

		userID := r.Header.Get("X-User-ID")
		if userID == "" {
			userID = "default-user"
		}

		cred := &models.Credential{
			Name:   req.Name,
			Kind:   req.Kind,
			Secret: req.Secret,
			UserID: userID,
		}

		created, err := h.store.CreateCredential(cred)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "Failed to create credential: %v", err)
			return
		}

		// SECURITY: Log without secret values
		log.Printf("[CREDENTIAL] Created %s credential '%s' (id=%s) — secrets stored securely",
			cred.Kind, cred.Name, created.ID)

		// Return safe projection — never return secrets
		writeJSON(w, http.StatusCreated, created.ToResponse())
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleCredentialByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/ai/credentials/")

	switch r.Method {
	case "GET":
		cred, err := h.store.GetCredential(id)
		if err != nil {
			writeError(w, http.StatusNotFound, "Credential not found: %v", err)
			return
		}
		// SECURITY: Never return secret values
		writeJSON(w, http.StatusOK, cred.ToResponse())
	case "DELETE":
		if err := h.store.DeleteCredential(id); err != nil {
			writeError(w, http.StatusNotFound, "Credential not found: %v", err)
			return
		}
		log.Printf("[CREDENTIAL] Deleted credential id=%s", id)
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleHealthCheck(w http.ResponseWriter, r *http.Request, connID string) {
	if r.Method != "GET" && r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	operationID := uuid.New().String()

	conn, err := h.store.GetConnection(connID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Connection not found: %v", err)
		return
	}

	// Get credential if the connection has one
	var cred *models.Credential
	if conn.CredentialID != "" {
		cred, err = h.store.GetCredential(conn.CredentialID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Associated credential not found: %v", err)
			return
		}
	}

	// Create provider and run health check
	provider, err := h.registry.Create(conn, cred)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to create provider: %v", err)
		return
	}

	status, err := provider.HealthCheck(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Health check failed: %v", err)
		return
	}
	status.OperationID = operationID

	// Update connection status based on health check
	switch status.Status {
	case "connected":
		_ = h.store.UpdateConnectionStatus(connID, models.StatusConnected)
	case "auth_failed":
		_ = h.store.UpdateConnectionStatus(connID, models.StatusError)
	case "unreachable":
		_ = h.store.UpdateConnectionStatus(connID, models.StatusDisconnected)
	}

	// SECURITY: Redact any secrets from health check messages
	if cred != nil {
		secrets := ai.CollectSecrets(cred.Secret)
		status.Message = ai.RedactSecrets(status.Message, secrets)
	}

	log.Printf("[HEALTH] Connection %s (%s): status=%s latency=%dms — operationId=%s",
		conn.Name, conn.Kind, status.Status, status.Latency, operationID)

	writeJSON(w, http.StatusOK, status)
}

func (h *Handler) handleGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req models.GenerationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: %v", err)
		return
	}

	if req.Prompt == "" {
		writeError(w, http.StatusBadRequest, "prompt is required")
		return
	}
	if req.ConnectionID == "" {
		writeError(w, http.StatusBadRequest, "connection_id is required")
		return
	}

	// Resolve connection
	conn, err := h.store.GetConnection(req.ConnectionID)
	if err != nil {
		writeError(w, http.StatusNotFound, "Connection not found: %v", err)
		return
	}

	// Resolve credential
	var cred *models.Credential
	if conn.CredentialID != "" {
		cred, err = h.store.GetCredential(conn.CredentialID)
		if err != nil {
			writeError(w, http.StatusBadRequest, "Credential not found: %v", err)
			return
		}
	}

	isSSE := r.Header.Get("Accept") == "text/event-stream"
	var progressChan chan string
	if isSSE {
		progressChan = make(chan string, 10)
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		if flusher, ok := w.(http.Flusher); ok {
			go func() {
				for msg := range progressChan {
					if strings.HasPrefix(msg, "TOKEN:") {
						token := strings.TrimPrefix(msg, "TOKEN:")
						jsonBytes, _ := json.Marshal(map[string]string{"token": token})
						fmt.Fprintf(w, "data: %s\n\n", string(jsonBytes))
					} else {
						fmt.Fprintf(w, "data: {\"status\": \"%s\"}\n\n", msg)
					}
					flusher.Flush()
				}
			}()
		} else {
			isSSE = false
			progressChan = nil
		}
	}

	// Execute generation pipeline
	response, err := h.pipeline.Execute(r.Context(), conn, cred, req.Prompt, progressChan)
	if progressChan != nil {
		close(progressChan)
	}

	if err != nil {
		if isSSE {
			fmt.Fprintf(w, "data: {\"error\": \"%v\"}\n\n", err)
			return
		}
		writeError(w, http.StatusInternalServerError, "Generation failed: %v", err)
		return
	}

	log.Printf("[GENERATE] Provider=%s model=%s success=%v latency=%dms — operationId=%s",
		response.ProviderKind, response.Model, response.Success,
		response.LatencyMs, response.OperationID)

	if isSSE {
		jsonBytes, _ := json.Marshal(response)
		fmt.Fprintf(w, "data: {\"result\": %s}\n\n", string(jsonBytes))
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (h *Handler) handleKanvasExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, "Failed to read body: %v", err)
		return
	}

	// Proxy to Meshery Server running on the host
	targetURL := "http://host.docker.internal:9081/api/pattern"
	proxyReq, err := http.NewRequest("POST", targetURL, bytes.NewBuffer(bodyBytes))
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to create proxy request: %v", err)
		return
	}

	proxyReq.Header.Set("Content-Type", "application/json")
	
	// Forward all cookies to ensure authentication succeeds
	for _, cookie := range r.Cookies() {
		proxyReq.AddCookie(cookie)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(proxyReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, "Failed to reach Meshery Server at 9081: %v", err)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func writeError(w http.ResponseWriter, status int, format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[ERROR] %s", msg)
	writeJSON(w, status, map[string]interface{}{
		"error":     msg,
		"timestamp": time.Now(),
	})
}

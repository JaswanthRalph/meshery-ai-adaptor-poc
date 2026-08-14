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

package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/ai"
	"github.com/meshery/ai-adapter-poc/internal/models"
	"github.com/meshery/ai-adapter-poc/internal/store"
)

func setupTestHandler() (*Handler, *http.ServeMux) {
	dataStore, err := store.New()
	if err != nil {
		panic("Failed to initialize store: " + err.Error())
	}
	registry := ai.NewRegistry()
	pipeline := ai.NewPipeline(registry)
	handler := NewHandler(dataStore, registry, pipeline)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)
	return handler, mux
}

func TestHandleProviders(t *testing.T) {
	_, mux := setupTestHandler()

	req, _ := http.NewRequest("GET", "/api/ai/providers", nil)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %v, got %v", http.StatusOK, rr.Code)
	}

	var providers []map[string]interface{}
	if err := json.Unmarshal(rr.Body.Bytes(), &providers); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if len(providers) == 0 {
		t.Error("Expected at least one provider")
	}
}

func TestHandleConnections(t *testing.T) {
	_, mux := setupTestHandler()

	// 1. Create a connection
	payload := map[string]interface{}{
		"name": "test-conn",
		"kind": models.ProviderOllama,
		"config": map[string]string{
			"model": "llama3",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/ai/connections", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status %v, got %v: %v", http.StatusCreated, rr.Code, rr.Body.String())
	}

	var conn models.Connection
	json.Unmarshal(rr.Body.Bytes(), &conn)

	if conn.Name != "test-conn" {
		t.Errorf("Expected name 'test-conn', got '%s'", conn.Name)
	}
	if conn.ID == "" {
		t.Error("Expected connection ID to be populated")
	}

	// 2. Get connection
	req, _ = http.NewRequest("GET", "/api/ai/connections/"+conn.ID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %v, got %v", http.StatusOK, rr.Code)
	}

	// 3. Delete connection
	req, _ = http.NewRequest("DELETE", "/api/ai/connections/"+conn.ID, nil)
	rr = httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("Expected status %v, got %v", http.StatusOK, rr.Code)
	}
}

func TestHandleCredentials(t *testing.T) {
	_, mux := setupTestHandler()

	// 1. Create a credential
	payload := map[string]interface{}{
		"name": "test-cred",
		"kind": models.ProviderOpenAI,
		"secret": map[string]string{
			"api_key": "sk-secret-123",
		},
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/ai/credentials", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("Expected status %v, got %v", http.StatusCreated, rr.Code)
	}

	var credResp models.CredentialResponse
	json.Unmarshal(rr.Body.Bytes(), &credResp)

	if credResp.Name != "test-cred" {
		t.Errorf("Expected name 'test-cred', got '%s'", credResp.Name)
	}
	// Verify secret is NOT exposed
	if val, ok := credResp.HasSecret["api_key"]; !ok || !val {
		t.Errorf("Expected HasSecret to contain 'api_key' as true")
	}
	if _, ok := credResp.HasSecret["password"]; ok {
		t.Errorf("Did not expect HasSecret to contain 'password'")
	}
}

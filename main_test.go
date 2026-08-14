package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/ai"
	"github.com/meshery/ai-adapter-poc/internal/handlers"
	"github.com/meshery/ai-adapter-poc/internal/models"
	"github.com/meshery/ai-adapter-poc/internal/store"
)

// mockProvider is a test provider that returns a valid Design.
type mockProvider struct{}

func (m *mockProvider) ID() models.ProviderKind { return "mock" }
func (m *mockProvider) Name() string            { return "Mock Provider" }

func (m *mockProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	return &models.HealthStatus{Status: "connected", ModelInfo: "mock-model"}, nil
}

func (m *mockProvider) Generate(ctx context.Context, input *ai.GenerateInput) (*ai.GenerateOutput, error) {
	return &ai.GenerateOutput{
		RawResponse: `{
			"name": "nginx-deployment",
			"schema_version": "designs.meshery.io/v1beta1",
			"version": "1.0.0",
			"components": [
				{
					"name": "nginx",
					"kind": "Deployment",
					"apiVersion": "apps/v1",
					"model": "kubernetes",
					"config": {"replicas": 3}
				}
			]
		}`,
		Model:      "mock-model",
		TokensUsed: 500,
	}, nil
}

func setupTestServer() (*httptest.Server, *store.Store) {
	dataStore := store.New()
	
	registry := ai.NewRegistry()
	// Register the mock provider to avoid hitting real APIs
	registry.Register("mock", func(conn *models.Connection, cred *models.Credential) (ai.Provider, error) {
		return &mockProvider{}, nil
	})

	pipeline := ai.NewPipeline(registry)

	handler := handlers.NewHandler(dataStore, registry, pipeline)
	mux := http.NewServeMux()
	handler.RegisterRoutes(mux)

	return httptest.NewServer(mux), dataStore
}

// TestServerEndToEnd tests the full adapter pipeline using an httptest server,
// mirroring the structure of server_test.go in the MCP repo. It ensures that
// the HTTP layer, the Store, the Pipeline, and the Registry integrate smoothly.
func TestServerEndToEnd(t *testing.T) {
	server, _ := setupTestServer()
	defer server.Close()

	client := server.Client()

	// 1. Create a credential
	credPayload := map[string]interface{}{
		"name": "mock-cred",
		"kind": "mock",
		"secret": map[string]string{
			"api_key": "sk-super-secret",
		},
	}
	credBody, _ := json.Marshal(credPayload)
	req, _ := http.NewRequest("POST", server.URL+"/api/ai/credentials", bytes.NewBuffer(credBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Failed to create credential: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created for credential, got %v", resp.StatusCode)
	}

	var cred models.CredentialResponse
	json.NewDecoder(resp.Body).Decode(&cred)
	resp.Body.Close()

	if cred.ID == "" {
		t.Fatal("Expected credential ID to be returned")
	}
	if val, ok := cred.SecretMasked["api_key"]; !ok || val == "sk-super-secret" {
		t.Fatalf("Secret leaked in response: %v", val)
	}

	// 2. Create a connection using that credential
	connPayload := map[string]interface{}{
		"name": "mock-conn",
		"kind": "mock",
		"config": map[string]string{
			"model": "mock-model",
		},
		"credential_id": cred.ID,
	}
	connBody, _ := json.Marshal(connPayload)
	req, _ = http.NewRequest("POST", server.URL+"/api/ai/connections", bytes.NewBuffer(connBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to create connection: %v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("Expected 201 Created for connection, got %v", resp.StatusCode)
	}

	var conn models.Connection
	json.NewDecoder(resp.Body).Decode(&conn)
	resp.Body.Close()

	// 3. Test Health Check
	req, _ = http.NewRequest("GET", server.URL+"/api/ai/connections/"+conn.ID+"/health", nil)
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to health check: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("Expected 200 OK for health check, got %v", resp.StatusCode)
	}

	var health models.HealthStatus
	json.NewDecoder(resp.Body).Decode(&health)
	resp.Body.Close()
	if health.Status != "connected" {
		t.Fatalf("Expected connected status, got %s", health.Status)
	}

	// 4. Test Generate
	genPayload := map[string]interface{}{
		"connection_id": conn.ID,
		"prompt":        "Deploy a simple nginx application",
	}
	genBody, _ := json.Marshal(genPayload)
	req, _ = http.NewRequest("POST", server.URL+"/api/ai/generate", bytes.NewBuffer(genBody))
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("Failed to generate: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("Expected 200 OK for generate, got %v. Body: %s", resp.StatusCode, string(b))
	}

	var genResp ai.PipelineResponse
	json.NewDecoder(resp.Body).Decode(&genResp)
	resp.Body.Close()

	if !genResp.Success {
		t.Fatalf("Expected generation to be successful. Validation errors: %v", genResp.ValidationErrors)
	}
	if genResp.Design == nil || genResp.Design.Name != "nginx-deployment" {
		t.Fatalf("Expected parsed Design with name 'nginx-deployment', got: %+v", genResp.Design)
	}

	// 5. Ensure Secrets never leaked into Generate raw output
	if strings.Contains(genResp.RawOutput, "sk-super-secret") {
		t.Fatal("SECURITY VIOLATION: Secret leaked in raw output of generate endpoint")
	}
}

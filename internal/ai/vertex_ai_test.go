package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

func TestNewVertexAIProvider(t *testing.T) {
	tests := []struct {
		name      string
		config    map[string]string
		cred      map[string]string
		expectErr bool
	}{
		{
			name: "valid config",
			config: map[string]string{
				"project_id": "test-project",
				"location":   "us-east1",
				"model":      "gemini-1.5-flash",
			},
			cred: map[string]string{
				"access_token": "ya29.validtoken",
			},
			expectErr: false,
		},
		{
			name: "missing project_id",
			config: map[string]string{
				"location": "us-east1",
			},
			cred: map[string]string{
				"access_token": "ya29.validtoken",
			},
			expectErr: true,
		},
		{
			name: "missing access token",
			config: map[string]string{
				"project_id": "test-project",
			},
			cred:      map[string]string{},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn := &models.Connection{Config: tt.config}
			var cred *models.Credential
			if tt.cred != nil {
				cred = &models.Credential{Secret: tt.cred}
			}

			p, err := NewVertexAIProvider(conn, cred)
			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if p.ProjectID != tt.config["project_id"] {
					t.Errorf("expected project %s, got %s", tt.config["project_id"], p.ProjectID)
				}
			}
		})
	}
}

func TestVertexAIHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if auth := r.Header.Get("Authorization"); auth != "Bearer valid-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"totalTokens": 5}`))
	}))
	defer server.Close()
	
	p := &VertexAIProvider{
		ProjectID:   "test",
		Location:    "us-central1",
		Model:       "gemini",
		AccessToken: "valid-token",
		BaseURL:     server.URL,
		client:      server.Client(),
	}

	ctx := context.Background()
	status, err := p.HealthCheck(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if status.Status != "connected" {
		t.Errorf("expected status connected, got %s", status.Status)
	}

	// Test auth fail
	p.AccessToken = "invalid-token"
	status, _ = p.HealthCheck(ctx)
	if status.Status != "auth_failed" {
		t.Errorf("expected auth_failed, got %s", status.Status)
	}
}

func TestVertexAIGenerateParsing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"candidates": [
				{
					"content": {
						"parts": [{"text": "{\"name\": \"test-design\"}"}]
					}
				}
			],
			"usageMetadata": {"totalTokenCount": 42}
		}`))
	}))
	defer server.Close()

	p := &VertexAIProvider{
		ProjectID:   "test",
		Location:    "us-central1",
		Model:       "gemini",
		AccessToken: "token",
		BaseURL:     server.URL,
		client:      server.Client(),
	}

	ctx := context.Background()
	out, err := p.Generate(ctx, &GenerateInput{UserPrompt: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if out.TokensUsed != 42 {
		t.Errorf("expected 42 tokens, got %d", out.TokensUsed)
	}
	if out.RawResponse != `{"name": "test-design"}` {
		t.Errorf("unexpected raw response: %s", out.RawResponse)
	}
}

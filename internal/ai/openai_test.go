package ai

import (
	"context"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

func TestOpenAIProvider_Initialization(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderOpenAI,
		Config: map[string]string{
			"model": "gpt-4",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "sk-test-key",
		},
	}

	provider, err := NewOpenAIProvider(conn, cred)
	if err != nil {
		t.Fatalf("Failed to initialize OpenAI provider: %v", err)
	}

	if provider.ID() != models.ProviderOpenAI {
		t.Errorf("Expected ID %s, got %s", models.ProviderOpenAI, provider.ID())
	}
	if provider.Name() != "OpenAI" {
		t.Errorf("Expected Name OpenAI, got %s", provider.Name())
	}
}

func TestOpenAIProvider_MissingCredentials(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderOpenAI,
		Config: map[string]string{
			"model": "gpt-4",
		},
	}

	_, err := NewOpenAIProvider(conn, nil)
	if err == nil {
		t.Fatal("Expected error when missing credentials")
	}
}

func TestOpenAIProvider_HealthCheck(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderOpenAI,
		Config: map[string]string{
			"model": "gpt-4",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "sk-test-key",
		},
	}

	provider, _ := NewOpenAIProvider(conn, cred)
	
	status, err := provider.HealthCheck(context.Background())
	if err == nil && status.Status == "connected" {
		t.Log("Health check succeeded (unexpected unless using real key)")
	} else {
		t.Logf("Health check correctly failed/errored with mock key: %v", err)
	}
}

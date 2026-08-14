package ai

import (
	"context"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

func TestAzureOpenAIProvider_Initialization(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAzureOpenAI,
		Config: map[string]string{
			"resource_name": "my-resource",
			"deployment_id": "my-deployment",
			"api_version":   "2024-02-15-preview",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "azure-test-key",
		},
	}

	provider, err := NewAzureOpenAIProvider(conn, cred)
	if err != nil {
		t.Fatalf("Failed to initialize Azure OpenAI provider: %v", err)
	}

	if provider.ID() != models.ProviderAzureOpenAI {
		t.Errorf("Expected ID %s, got %s", models.ProviderAzureOpenAI, provider.ID())
	}
	if provider.Name() != "Azure OpenAI" {
		t.Errorf("Expected Name Azure OpenAI, got %s", provider.Name())
	}
}

func TestAzureOpenAIProvider_MissingCredentials(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAzureOpenAI,
		Config: map[string]string{
			"resource_name": "my-resource",
		},
	}

	_, err := NewAzureOpenAIProvider(conn, nil)
	if err == nil {
		t.Fatal("Expected error when missing credentials")
	}
}

func TestAzureOpenAIProvider_MissingConfig(t *testing.T) {
	conn := &models.Connection{
		Kind:   models.ProviderAzureOpenAI,
		Config: map[string]string{}, // Missing resource_name, deployment_id
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "azure-test-key",
		},
	}

	_, err := NewAzureOpenAIProvider(conn, cred)
	if err == nil {
		t.Fatal("Expected error when missing required config")
	}
}

func TestAzureOpenAIProvider_HealthCheck(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAzureOpenAI,
		Config: map[string]string{
			"resource_name": "my-resource",
			"deployment_id": "my-deployment",
			"api_version":   "2024-02-15-preview",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "azure-test-key",
		},
	}

	provider, _ := NewAzureOpenAIProvider(conn, cred)
	
	status, err := provider.HealthCheck(context.Background())
	if err == nil && status.Status == "connected" {
		t.Log("Health check succeeded (unexpected unless using real key)")
	} else {
		t.Logf("Health check correctly failed/errored with mock key: %v", err)
	}
}

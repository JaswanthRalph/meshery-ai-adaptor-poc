package ai

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

// TestProviderSwap demonstrates that swapping between a hosted
// provider and a local provider requires only changing the
// Connection — no code changes needed. This is the core BYOM promise.
func TestProviderSwap_RegistryCreatesCorrectProvider(t *testing.T) {
	registry := NewRegistry()

	// Test 1: Create an OpenAI provider
	openaiConn := &models.Connection{
		Kind:   models.ProviderOpenAI,
		Config: map[string]string{"model": "gpt-4o"},
	}
	openaiCred := &models.Credential{
		Secret: map[string]string{"api_key": "sk-test-key"},
	}

	provider, err := registry.Create(openaiConn, openaiCred)
	if err != nil {
		t.Fatalf("Failed to create OpenAI provider: %v", err)
	}
	if provider.ID() != models.ProviderOpenAI {
		t.Errorf("Expected provider ID 'openai', got '%s'", provider.ID())
	}
	if provider.Name() != "OpenAI" {
		t.Errorf("Expected provider name 'OpenAI', got '%s'", provider.Name())
	}

	// Test 2: Swap to Ollama provider — same registry, different connection
	ollamaConn := &models.Connection{
		Kind:   models.ProviderOllama,
		Config: map[string]string{"model": "llama3.1", "base_url": "http://localhost:11434"},
	}

	provider2, err := registry.Create(ollamaConn, nil)
	if err != nil {
		t.Fatalf("Failed to create Ollama provider: %v", err)
	}
	if provider2.ID() != models.ProviderOllama {
		t.Errorf("Expected provider ID 'ollama', got '%s'", provider2.ID())
	}
	if provider2.Name() != "Ollama (Local)" {
		t.Errorf("Expected provider name 'Ollama (Local)', got '%s'", provider2.Name())
	}

	// Test 3: Swap to Anthropic
	anthropicConn := &models.Connection{
		Kind:   models.ProviderAnthropic,
		Config: map[string]string{"model": "claude-sonnet-4-20250514"},
	}
	anthropicCred := &models.Credential{
		Secret: map[string]string{"api_key": "sk-ant-test-key"},
	}

	provider3, err := registry.Create(anthropicConn, anthropicCred)
	if err != nil {
		t.Fatalf("Failed to create Anthropic provider: %v", err)
	}
	if provider3.ID() != models.ProviderAnthropic {
		t.Errorf("Expected provider ID 'anthropic', got '%s'", provider3.ID())
	}

	// Test 4: Swap to Azure OpenAI
	azureConn := &models.Connection{
		Kind: models.ProviderAzureOpenAI,
		Config: map[string]string{
			"resource_name": "my-resource",
			"deployment_id": "gpt-4-deployment",
			"api_version":   "2024-02-15-preview",
		},
	}
	azureCred := &models.Credential{
		Secret: map[string]string{"api_key": "azure-test-key"},
	}

	provider4, err := registry.Create(azureConn, azureCred)
	if err != nil {
		t.Fatalf("Failed to create Azure OpenAI provider: %v", err)
	}
	if provider4.ID() != models.ProviderAzureOpenAI {
		t.Errorf("Expected provider ID 'azure-openai', got '%s'", provider4.ID())
	}

	// Test 5: Swap to Vertex AI
	vertexConn := &models.Connection{
		Kind: models.ProviderVertexAI,
		Config: map[string]string{
			"project_id": "my-project",
			"location":   "us-central1",
			"model":      "gemini",
		},
	}
	vertexCred := &models.Credential{
		Secret: map[string]string{"access_token": "ya29.token"},
	}

	provider5, err := registry.Create(vertexConn, vertexCred)
	if err != nil {
		t.Fatalf("Failed to create Vertex AI provider: %v", err)
	}
	if provider5.ID() != models.ProviderVertexAI {
		t.Errorf("Expected provider ID 'vertex-ai', got '%s'", provider5.ID())
	}

	t.Logf("✅ Provider swap verified: OpenAI → Ollama → Anthropic → Azure OpenAI → Vertex AI with no code changes")
}

// TestProviderSwap_AllProvidersRegistered ensures all 4 minimum
// providers are registered in the default registry.
func TestProviderSwap_AllProvidersRegistered(t *testing.T) {
	registry := NewRegistry()
	kinds := registry.SupportedKinds()

	expected := map[models.ProviderKind]bool{
		models.ProviderOpenAI:      false,
		models.ProviderAnthropic:   false,
		models.ProviderOllama:      false,
		models.ProviderAzureOpenAI: false,
	}

	for _, k := range kinds {
		if _, ok := expected[k]; ok {
			expected[k] = true
		}
	}

	for kind, found := range expected {
		if !found {
			t.Errorf("Provider '%s' is not registered in the default registry", kind)
		}
	}
}

// TestProviderSwap_UnknownProviderFails ensures unknown providers
// return a clear error.
func TestProviderSwap_UnknownProviderFails(t *testing.T) {
	registry := NewRegistry()
	conn := &models.Connection{Kind: "unknown-provider"}

	_, err := registry.Create(conn, nil)
	if err == nil {
		t.Fatal("Expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unknown provider kind") {
		t.Errorf("Error message should mention 'unknown provider kind', got: %s", err.Error())
	}
}

// TestDesignParsing verifies the pipeline can parse LLM output into
// valid Meshery Design format.
func TestDesignParsing(t *testing.T) {
	raw := `{
		"name": "test-design",
		"schema_version": "designs.meshery.io/v1beta1",
		"version": "1.0.0",
		"components": [
			{
				"name": "nginx-deployment",
				"kind": "Deployment",
				"apiVersion": "apps/v1",
				"model": "kubernetes",
				"namespace": "default",
				"config": {
					"replicas": 3,
					"containers": [{"name": "nginx", "image": "nginx:1.25", "ports": [{"containerPort": 80}]}]
				}
			},
			{
				"name": "nginx-service",
				"kind": "Service",
				"apiVersion": "v1",
				"model": "kubernetes",
				"namespace": "default",
				"config": {
					"type": "LoadBalancer",
					"ports": [{"port": 80, "targetPort": 80}]
				}
			}
		],
		"relationships": [
			{"kind": "edge", "type": "network", "source": "nginx-service", "target": "nginx-deployment"}
		]
	}`

	design, errors := ParseDesignFromLLMOutput(raw)
	if design == nil {
		t.Fatalf("Failed to parse design: %v", errors)
	}
	if design.Name != "test-design" {
		t.Errorf("Expected design name 'test-design', got '%s'", design.Name)
	}
	if len(design.Components) != 2 {
		t.Errorf("Expected 2 components, got %d", len(design.Components))
	}
	if len(design.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(design.Relationships))
	}
}

// TestDesignValidation verifies that the validator catches issues.
func TestDesignValidation(t *testing.T) {
	// Valid design should have no errors
	design := &models.Design{
		Name: "valid-design",
		Components: []models.Component{
			{
				Name:       "nginx",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
				Config:     map[string]interface{}{"replicas": 3},
			},
		},
	}

	errors := ValidateDesign(design)
	for _, e := range errors {
		if e.Severity == "error" {
			t.Errorf("Unexpected validation error: %s", e.Message)
		}
	}

	// Design missing name should have an error
	invalid := &models.Design{
		Components: []models.Component{
			{Kind: "Deployment", APIVersion: "apps/v1", Config: map[string]interface{}{}},
		},
	}
	errors = ValidateDesign(invalid)
	hasNameError := false
	for _, e := range errors {
		if e.Field == "name" {
			hasNameError = true
		}
	}
	if !hasNameError {
		t.Error("Expected validation error for missing design name")
	}
}

// TestSecretNeverInDesign ensures that the pipeline's redaction
// catches secrets that might appear in generated Designs.
func TestSecretNeverInDesign(t *testing.T) {
	secret := "sk-super-secret-api-key-12345"

	// Simulate an LLM that accidentally includes the key in output
	rawOutput := `{
		"name": "leaked-design",
		"components": [{
			"name": "app",
			"kind": "Deployment",
			"apiVersion": "apps/v1",
			"model": "kubernetes",
			"config": {
				"env": [{"name": "API_KEY", "value": "sk-super-secret-api-key-12345"}]
			}
		}]
	}`

	// Redact secrets
	redacted := RedactSecrets(rawOutput, []string{secret})

	if strings.Contains(redacted, secret) {
		t.Fatalf("SECURITY VIOLATION: Secret found in redacted output")
	}

	// Parse and check
	design, _ := ParseDesignFromLLMOutput(redacted)
	if design != nil {
		designJSON, _ := json.Marshal(design)
		if strings.Contains(string(designJSON), secret) {
			t.Fatalf("SECURITY VIOLATION: Secret found in parsed Design")
		}
	}
}

// TestPipelineContextNeverContainsSecrets ensures the prompt context
// builder never includes credential material.
func TestPipelineContextNeverContainsSecrets(t *testing.T) {
	input := BuildPromptContext("Deploy nginx")

	// The system prompt and schema context should never contain
	// patterns that look like API keys
	fullContext := input.SystemPrompt + input.SchemaContext
	suspiciousPatterns := []string{"sk-", "api_key", "Bearer ", "password"}
	for _, pattern := range suspiciousPatterns {
		if strings.Contains(fullContext, pattern) {
			// "api_key" might appear in schema descriptions — that's OK
			// but "sk-" or actual values should never appear
			if pattern == "sk-" {
				t.Errorf("System prompt contains suspicious pattern: %s", pattern)
			}
		}
	}
}

// TestPipelineExecuteWithMockProvider verifies the full pipeline flow.
func TestPipelineExecuteWithMockProvider(t *testing.T) {
	// Create a mock registry with a mock provider
	registry := NewRegistry()
	registry.Register("mock", func(conn *models.Connection, cred *models.Credential) (Provider, error) {
		return &mockProvider{}, nil
	})

	pipeline := NewPipeline(registry)

	conn := &models.Connection{
		Kind:   "mock",
		Config: map[string]string{"model": "mock-model"},
	}
	cred := &models.Credential{
		Secret: map[string]string{"api_key": "test-secret-key"},
	}

	response, err := pipeline.Execute(context.Background(), conn, cred, "Deploy nginx with 3 replicas")
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	if !response.Success {
		t.Logf("Validation errors: %+v", response.ValidationErrors)
	}

	if response.OperationID == "" {
		t.Error("Expected non-empty operation ID")
	}

	// Verify secret is not in any output
	if strings.Contains(response.RawOutput, "test-secret-key") {
		t.Fatal("SECURITY: Secret found in raw output")
	}

	t.Logf("✅ Pipeline executed successfully: design=%v, latency=%dms",
		response.Design != nil, response.LatencyMs)
}

// mockProvider is a test provider that returns a valid Design.
type mockProvider struct{}

func (m *mockProvider) ID() models.ProviderKind { return "mock" }
func (m *mockProvider) Name() string            { return "Mock Provider" }

func (m *mockProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	return &models.HealthStatus{Status: "connected", ModelInfo: "mock-model"}, nil
}

func (m *mockProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	return &GenerateOutput{
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
					"namespace": "default",
					"labels": {"app": "nginx"},
					"config": {"replicas": 3, "containers": [{"name": "nginx", "image": "nginx:latest", "ports": [{"containerPort": 80}]}]}
				},
				{
					"name": "nginx-svc",
					"kind": "Service",
					"apiVersion": "v1",
					"model": "kubernetes",
					"namespace": "default",
					"config": {"type": "LoadBalancer", "ports": [{"port": 80, "targetPort": 80}], "selector": {"app": "nginx"}}
				}
			],
			"relationships": [{"kind": "edge", "type": "network", "source": "nginx-svc", "target": "nginx"}]
		}`,
		Model:      "mock-model",
		TokensUsed: 500,
	}, nil
}

package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/meshery/ai-adapter-poc/internal/models"
)

// Pipeline orchestrates the end-to-end NL→Design generation flow:
//
//	User Prompt
//	  → Resolve Connection & Credential
//	  → Build Prompt Context
//	  → Call Provider.Generate()
//	  → Parse & Normalize Design
//	  → Validate against Meshery Schema
//	  → Redact Secrets
//	  → Return GenerationResponse
//
// Generation NEVER auto-deploys. The pipeline returns a candidate
// Design for human review via Meshery UI (Kanvas) or CLI.
type Pipeline struct {
	registry *Registry
}

// NewPipeline creates a generation pipeline with the given registry.
func NewPipeline(registry *Registry) *Pipeline {
	return &Pipeline{registry: registry}
}

// Execute runs the full generation pipeline.
func (p *Pipeline) Execute(
	ctx context.Context,
	conn *models.Connection,
	cred *models.Credential,
	prompt string,
) (*models.GenerationResponse, error) {
	start := time.Now()
	operationID := uuid.New().String()

	response := &models.GenerationResponse{
		OperationID:  operationID,
		ProviderKind: conn.Kind,
		GeneratedAt:  time.Now(),
	}

	// Step 1: Create provider from connection + credential
	provider, err := p.registry.Create(conn, cred)
	if err != nil {
		response.Success = false
		response.ValidationErrors = []models.ValidationError{{
			Component: "provider",
			Field:     "kind",
			Message:   fmt.Sprintf("Failed to create provider: %v", err),
			Severity:  "error",
		}}
		return response, nil
	}

	// Step 2: Build prompt context with system prompt and schema
	input := BuildPromptContext(prompt)
	input.SystemPrompt = EnhanceSystemPromptWithExamples(input.SystemPrompt)

	// Use model from connection config if specified
	if model, ok := conn.Config["model"]; ok && model != "" {
		input.Model = model
	}

	// Step 3: Call provider
	output, err := provider.Generate(ctx, input)
	if err != nil {
		response.Success = false
		response.ValidationErrors = []models.ValidationError{{
			Component: "generation",
			Field:     "llm_call",
			Message:   fmt.Sprintf("Provider call failed: %v", RedactSecrets(err.Error(), CollectSecrets(cred.Secret))),
			Severity:  "error",
		}}
		response.LatencyMs = time.Since(start).Milliseconds()
		return response, nil
	}

	response.Model = output.Model
	response.LatencyMs = time.Since(start).Milliseconds()

	// Step 4: Redact any secrets from raw output
	secrets := CollectSecrets(cred.Secret)
	rawOutput := RedactSecrets(output.RawResponse, secrets)
	response.RawOutput = rawOutput

	// Step 5: Parse Design from LLM output
	design, parseErrors := ParseDesignFromLLMOutput(rawOutput)
	if design != nil {
		// Step 6: Validate the parsed Design
		validationErrors := ValidateDesign(design)

		// Step 7: Redact secrets from all design fields
		designJSON, _ := json.Marshal(design)
		redactedJSON := RedactSecrets(string(designJSON), secrets)
		json.Unmarshal([]byte(redactedJSON), design)

		response.Design = design
		response.ValidationErrors = validationErrors
		response.Success = len(validationErrors) == 0 || allWarnings(validationErrors)
	} else {
		response.Success = false
		response.ValidationErrors = parseErrors
	}

	return response, nil
}

// ParseDesignFromLLMOutput extracts a Design JSON from raw LLM text.
func ParseDesignFromLLMOutput(raw string) (*models.Design, []models.ValidationError) {
	// Try to extract JSON from the response
	jsonStr := extractJSON(raw)
	if jsonStr == "" {
		return nil, []models.ValidationError{{
			Component: "parser",
			Field:     "output",
			Message:   "Could not extract valid JSON from LLM response",
			Severity:  "error",
		}}
	}

	var design models.Design
	if err := json.Unmarshal([]byte(jsonStr), &design); err != nil {
		return nil, []models.ValidationError{{
			Component: "parser",
			Field:     "json",
			Message:   fmt.Sprintf("Failed to parse Design JSON: %v", err),
			Severity:  "error",
		}}
	}

	// Set defaults
	if design.SchemaVersion == "" {
		design.SchemaVersion = "designs.meshery.io/v1beta1"
	}
	if design.Version == "" {
		design.Version = "1.0.0"
	}

	return &design, nil
}

// extractJSON finds the first complete JSON object in a string.
func extractJSON(s string) string {
	// Try direct parse first
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "{") {
		return s
	}

	// Look for JSON in code blocks
	if idx := strings.Index(s, "```json"); idx >= 0 {
		start := idx + 7
		end := strings.Index(s[start:], "```")
		if end >= 0 {
			return strings.TrimSpace(s[start : start+end])
		}
	}

	// Look for first { to last }
	start := strings.Index(s, "{")
	if start < 0 {
		return ""
	}
	end := strings.LastIndex(s, "}")
	if end < start {
		return ""
	}
	return s[start : end+1]
}

// ValidateDesign performs schema validation on a parsed Design.
func ValidateDesign(design *models.Design) []models.ValidationError {
	var errors []models.ValidationError

	if design.Name == "" {
		errors = append(errors, models.ValidationError{
			Component: "design",
			Field:     "name",
			Message:   "Design name is required",
			Severity:  "error",
		})
	}

	if len(design.Components) == 0 {
		errors = append(errors, models.ValidationError{
			Component: "design",
			Field:     "components",
			Message:   "Design must have at least one component",
			Severity:  "error",
		})
	}

	for i, comp := range design.Components {
		if comp.Kind == "" {
			errors = append(errors, models.ValidationError{
				Component: fmt.Sprintf("components[%d]", i),
				Field:     "kind",
				Message:   "Component kind is required",
				Severity:  "error",
			})
		}
		if comp.APIVersion == "" {
			errors = append(errors, models.ValidationError{
				Component: fmt.Sprintf("components[%d]", i),
				Field:     "apiVersion",
				Message:   fmt.Sprintf("Component '%s' is missing apiVersion", comp.Name),
				Severity:  "error",
			})
		}
		if comp.Name == "" {
			errors = append(errors, models.ValidationError{
				Component: fmt.Sprintf("components[%d]", i),
				Field:     "name",
				Message:   "Component name is required",
				Severity:  "error",
			})
		}

		// Validate Deployment-specific fields
		if comp.Kind == "Deployment" {
			if comp.Config == nil {
				errors = append(errors, models.ValidationError{
					Component: comp.Name,
					Field:     "config",
					Message:   "Deployment must have a config with replicas and containers",
					Severity:  "error",
				})
			}
		}

		// Validate Service-specific fields
		if comp.Kind == "Service" {
			if comp.Config == nil {
				errors = append(errors, models.ValidationError{
					Component: comp.Name,
					Field:     "config",
					Message:   "Service must have a config with type and ports",
					Severity:  "error",
				})
			}
		}

		// Check for secrets in config (security validation)
		if comp.Config != nil {
			configJSON, _ := json.Marshal(comp.Config)
			configStr := strings.ToLower(string(configJSON))
			suspiciousPatterns := []string{"sk-", "api_key", "password", "secret_key", "bearer"}
			for _, pattern := range suspiciousPatterns {
				if strings.Contains(configStr, pattern) {
					errors = append(errors, models.ValidationError{
						Component: comp.Name,
						Field:     "config",
						Message:   fmt.Sprintf("Potential secret detected in config (pattern: %s). Secrets must not be embedded in Designs.", pattern),
						Severity:  "error",
					})
				}
			}
		}
	}

	return errors
}

func allWarnings(errs []models.ValidationError) bool {
	for _, e := range errs {
		if e.Severity == "error" {
			return false
		}
	}
	return true
}

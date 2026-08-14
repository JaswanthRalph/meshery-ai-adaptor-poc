// Package ai implements the BYOM (Bring Your Own Model) abstraction layer
// for Meshery's AI-driven Design generation. It defines the Provider
// interface that all AI/LLM backends must satisfy, enabling seamless
// provider swapping with no code changes.
package ai

import (
	"context"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

// Provider is the core abstraction for all AI/LLM backends.
// Each provider implementation wraps a specific API (OpenAI, Anthropic,
// Ollama, Azure OpenAI) behind this uniform interface.
//
// Design principle: a user can swap from OpenAI to Ollama by changing
// only their Connection configuration — no code changes required.
type Provider interface {
	// ID returns the provider kind identifier (e.g., "openai", "ollama").
	ID() models.ProviderKind

	// Name returns a human-readable display name for the provider.
	Name() string

	// HealthCheck verifies that the connection is reachable and the
	// credential is valid. Returns a HealthStatus with latency,
	// model info, and any error details.
	HealthCheck(ctx context.Context) (*models.HealthStatus, error)

	// Generate sends a prompt with schema context to the LLM and
	// returns a candidate Design. The implementation must:
	//   1. Never include credential material in the prompt
	//   2. Never include credential material in the response
	//   3. Parse the LLM output into a structured Design
	Generate(ctx context.Context, req *GenerateInput) (*GenerateOutput, error)
}

// GenerateInput is the internal input to a Provider's Generate method.
// It contains the user prompt and pre-built context (system prompt,
// schema snippets) that the provider sends to the LLM.
type GenerateInput struct {
	UserPrompt    string
	SystemPrompt  string
	SchemaContext string
	Model         string // Override model if specified in connection config
}

// GenerateOutput is the internal output from a Provider's Generate method.
type GenerateOutput struct {
	RawResponse string          // The raw LLM text response
	Design      *models.Design  // Parsed Design (may be nil if parsing fails)
	Model       string          // The actual model used
	TokensUsed  int             // Approximate token usage
}

// Package models defines the core data types for Meshery AI provider
// connections and credentials. These mirror Meshery's first-class
// Connection and Credential constructs, ensuring AI providers are
// managed identically to Kubernetes clusters, Prometheus, etc.
package models

import (
	"time"
)

// ConnectionStatus represents the lifecycle state of a connection.
type ConnectionStatus string

const (
	StatusDiscovered  ConnectionStatus = "discovered"
	StatusRegistered  ConnectionStatus = "registered"
	StatusConnected   ConnectionStatus = "connected"
	StatusDisconnected ConnectionStatus = "disconnected"
	StatusNotFound    ConnectionStatus = "not_found"
	StatusError       ConnectionStatus = "error"
)

// ProviderKind identifies the AI provider type.
type ProviderKind string

const (
	ProviderOpenAI      ProviderKind = "openai"
	ProviderAnthropic   ProviderKind = "anthropic"
	ProviderOllama      ProviderKind = "ollama"
	ProviderAzureOpenAI ProviderKind = "azure-openai"
	ProviderVertexAI    ProviderKind = "vertex-ai"
)

// Connection represents a registered AI provider connection.
// It is the Meshery Connection construct applied to AI/LLM backends.
type Connection struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Kind         ProviderKind      `json:"kind"`
	Status       ConnectionStatus  `json:"status"`
	Config       map[string]string `json:"config"`       // Provider-specific config (base_url, model, etc.)
	CredentialID string            `json:"credential_id"` // Reference to associated Credential
	UserID       string            `json:"user_id"`       // Owner - connections are per-user
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

// Credential stores encrypted authentication material.
// Secrets are never returned to API clients; the Secret field
// is omitted from all JSON serialization.
type Credential struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Kind      ProviderKind      `json:"kind"`
	UserID    string            `json:"user_id"`
	Secret    map[string]string `json:"-"`          // NEVER serialized to JSON
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// CredentialResponse is the safe projection of a Credential
// returned to API clients. Secrets are replaced with existence flags.
type CredentialResponse struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Kind      ProviderKind      `json:"kind"`
	UserID    string            `json:"user_id"`
	HasSecret map[string]bool   `json:"has_secret"` // e.g., {"api_key": true}
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// ToResponse converts a Credential to a safe CredentialResponse,
// stripping secrets and replacing them with existence flags.
func (c *Credential) ToResponse() CredentialResponse {
	hasSecret := make(map[string]bool)
	for k := range c.Secret {
		hasSecret[k] = true
	}
	return CredentialResponse{
		ID:        c.ID,
		Name:      c.Name,
		Kind:      c.Kind,
		UserID:    c.UserID,
		HasSecret: hasSecret,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}

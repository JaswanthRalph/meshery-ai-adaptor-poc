package models

import "time"

// GenerationRequest is the API payload for NL→Design generation.
type GenerationRequest struct {
	Prompt       string `json:"prompt"`
	ConnectionID string `json:"connection_id"`
	UserID       string `json:"user_id,omitempty"` // Populated by auth middleware
}

// GenerationResponse is the API response containing either a
// candidate Design or structured validation errors.
type GenerationResponse struct {
	Success          bool               `json:"success"`
	OperationID      string             `json:"operation_id"`
	Design           *Design            `json:"design,omitempty"`
	ValidationErrors []ValidationError  `json:"validation_errors,omitempty"`
	RawOutput        string             `json:"raw_output,omitempty"` // For debugging; secrets redacted
	ProviderKind     ProviderKind       `json:"provider_kind"`
	Model            string             `json:"model_used"`
	GeneratedAt      time.Time          `json:"generated_at"`
	LatencyMs        int64              `json:"latency_ms"`
}

// Design represents a Meshery Design document — the output of
// the NL→Infrastructure generation pipeline.
type Design struct {
	Name         string           `json:"name"`
	SchemaVersion string          `json:"schema_version"`
	Version      string           `json:"version"`
	Components   []Component      `json:"components"`
	Relationships []Relationship  `json:"relationships,omitempty"`
}

// Component represents a single infrastructure component within a Design
// (e.g., a Kubernetes Deployment, Service, ConfigMap).
type Component struct {
	Name       string                 `json:"name"`
	Kind       string                 `json:"kind"`
	APIVersion string                 `json:"apiVersion"`
	Model      string                 `json:"model"`
	Namespace  string                 `json:"namespace,omitempty"`
	Labels     map[string]string      `json:"labels,omitempty"`
	Config     map[string]interface{} `json:"config"`
}

// Relationship defines how components connect to each other.
type Relationship struct {
	Kind   string `json:"kind"`   // e.g., "edge", "hierarchical"
	Type   string `json:"type"`   // e.g., "network", "mount", "parent"
	Source string `json:"source"` // Component name
	Target string `json:"target"` // Component name
}

// ValidationError represents a per-component schema validation error.
type ValidationError struct {
	Component string `json:"component"`
	Field     string `json:"field"`
	Message   string `json:"message"`
	Severity  string `json:"severity"` // "error" or "warning"
}

// HealthStatus represents the result of a provider connectivity check.
type HealthStatus struct {
	Status      string `json:"status"`       // connected, unreachable, auth_failed, rate_limited
	Latency     int64  `json:"latency_ms"`
	ModelInfo   string `json:"model_info"`
	Message     string `json:"message,omitempty"`
	OperationID string `json:"operation_id"`
	CheckedAt   time.Time `json:"checked_at"`
}

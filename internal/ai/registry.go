package ai

import (
	"fmt"
	"sync"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

// ProviderFactory creates a Provider from a Connection and Credential.
type ProviderFactory func(conn *models.Connection, cred *models.Credential) (Provider, error)

// Registry manages provider factories and instantiates providers
// at runtime. It is the central dispatch table for the BYOM system:
// new provider kinds are registered here, and the Generation Pipeline
// looks up providers by kind.
type Registry struct {
	mu        sync.RWMutex
	factories map[models.ProviderKind]ProviderFactory
}

// NewRegistry creates a Registry pre-loaded with all built-in providers.
func NewRegistry() *Registry {
	r := &Registry{
		factories: make(map[models.ProviderKind]ProviderFactory),
	}
	// Register all built-in providers
	r.Register(models.ProviderOpenAI, NewOpenAIProvider)
	r.Register(models.ProviderAnthropic, NewAnthropicProvider)
	r.Register(models.ProviderOllama, NewOllamaProvider)
	r.Register(models.ProviderAzureOpenAI, NewAzureOpenAIProvider)
	return r
}

// Register adds a provider factory for the given kind.
func (r *Registry) Register(kind models.ProviderKind, factory ProviderFactory) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.factories[kind] = factory
}

// Create instantiates a Provider for the given Connection and Credential.
func (r *Registry) Create(conn *models.Connection, cred *models.Credential) (Provider, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	factory, ok := r.factories[conn.Kind]
	if !ok {
		return nil, fmt.Errorf("unknown provider kind: %s", conn.Kind)
	}
	return factory(conn, cred)
}

// SupportedKinds returns all registered provider kinds.
func (r *Registry) SupportedKinds() []models.ProviderKind {
	r.mu.RLock()
	defer r.mu.RUnlock()

	kinds := make([]models.ProviderKind, 0, len(r.factories))
	for k := range r.factories {
		kinds = append(kinds, k)
	}
	return kinds
}

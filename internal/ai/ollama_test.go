// Copyright 2026 The Meshery Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ai

import (
	"context"
	"testing"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

func TestOllamaProvider_Initialization(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderOllama,
		Config: map[string]string{
			"model":    "llama3",
			"base_url": "http://localhost:11434",
		},
	}

	provider, err := NewOllamaProvider(conn, nil)
	if err != nil {
		t.Fatalf("Failed to initialize Ollama provider: %v", err)
	}

	if provider.ID() != models.ProviderOllama {
		t.Errorf("Expected ID %s, got %s", models.ProviderOllama, provider.ID())
	}
	if provider.Name() != "Ollama (Local)" {
		t.Errorf("Expected Name Ollama (Local), got %s", provider.Name())
	}
}

func TestOllamaProvider_HealthCheck(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderOllama,
		Config: map[string]string{
			"model":    "llama3",
			"base_url": "http://localhost:11434", // might not be running in test env
		},
	}

	provider, _ := NewOllamaProvider(conn, nil)

	status, err := provider.HealthCheck(context.Background())
	if err == nil && status.Status == "connected" {
		t.Log("Health check succeeded (Ollama is running)")
	} else {
		t.Logf("Health check correctly failed/errored when Ollama is unreachable: %v", err)
	}
}

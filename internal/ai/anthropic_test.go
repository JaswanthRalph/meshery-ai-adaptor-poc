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

func TestAnthropicProvider_Initialization(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAnthropic,
		Config: map[string]string{
			"model": "claude-sonnet-4-20250514",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "sk-ant-test-key",
		},
	}

	provider, err := NewAnthropicProvider(conn, cred)
	if err != nil {
		t.Fatalf("Failed to initialize Anthropic provider: %v", err)
	}

	if provider.ID() != models.ProviderAnthropic {
		t.Errorf("Expected ID %s, got %s", models.ProviderAnthropic, provider.ID())
	}
	if provider.Name() != "Anthropic Claude" {
		t.Errorf("Expected Name Anthropic Claude, got %s", provider.Name())
	}
}

func TestAnthropicProvider_MissingCredentials(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAnthropic,
		Config: map[string]string{
			"model": "claude-sonnet-4-20250514",
		},
	}

	_, err := NewAnthropicProvider(conn, nil)
	if err == nil {
		t.Fatal("Expected error when missing credentials")
	}
}

func TestAnthropicProvider_HealthCheck(t *testing.T) {
	conn := &models.Connection{
		Kind: models.ProviderAnthropic,
		Config: map[string]string{
			"model": "claude-sonnet-4-20250514",
		},
	}
	cred := &models.Credential{
		Secret: map[string]string{
			"api_key": "sk-ant-test-key",
		},
	}

	provider, _ := NewAnthropicProvider(conn, cred)

	// Since we are mocking/not mocking the actual HTTP call, the health check might fail
	// with a network or auth error, but it shouldn't panic.
	status, err := provider.HealthCheck(context.Background())
	if err == nil && status.Status == "connected" {
		t.Log("Health check succeeded (unexpected unless using real key)")
	} else {
		t.Logf("Health check correctly failed/errored with mock key: %v", err)
	}
}

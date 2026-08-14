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
	"strings"
	"testing"
)

func TestRedactSecrets_BasicRedaction(t *testing.T) {
	secret := "sk-proj-abc123xyz789"
	input := "Using API key sk-proj-abc123xyz789 to connect"
	result := RedactSecrets(input, []string{secret})
	if strings.Contains(result, secret) {
		t.Errorf("Secret was not redacted from output: %s", result)
	}
	if !strings.Contains(result, "[REDACTED]") {
		t.Errorf("Expected [REDACTED] in output: %s", result)
	}
}

func TestRedactSecrets_MultipleSecrets(t *testing.T) {
	secrets := []string{"sk-key-one", "sk-key-two"}
	input := "Key 1: sk-key-one, Key 2: sk-key-two"
	result := RedactSecrets(input, secrets)
	for _, s := range secrets {
		if strings.Contains(result, s) {
			t.Errorf("Secret %q was not redacted", s)
		}
	}
}

func TestRedactSecrets_EmptyInput(t *testing.T) {
	result := RedactSecrets("", []string{"secret"})
	if result != "" {
		t.Errorf("Expected empty string, got: %s", result)
	}
}

func TestRedactSecrets_NoSecrets(t *testing.T) {
	input := "No secrets here"
	result := RedactSecrets(input, nil)
	if result != input {
		t.Errorf("Expected unchanged input, got: %s", result)
	}
}

func TestRedactSecrets_SecretInJSON(t *testing.T) {
	secret := "sk-test-secret-key-12345"
	input := `{"api_key": "sk-test-secret-key-12345", "model": "gpt-4"}`
	result := RedactSecrets(input, []string{secret})
	if strings.Contains(result, secret) {
		t.Errorf("Secret was not redacted from JSON: %s", result)
	}
}

func TestRedactSecrets_SecretInDesignYAML(t *testing.T) {
	secret := "my-super-secret-api-key-value"
	input := `components:
  - name: my-app
    config:
      env:
        - name: API_KEY
          value: my-super-secret-api-key-value`
	result := RedactSecrets(input, []string{secret})
	if strings.Contains(result, secret) {
		t.Errorf("Secret leaked into Design YAML: %s", result)
	}
}

func TestRedactSecrets_PartialKeyMatch(t *testing.T) {
	secret := "sk-proj-abcdefghijklmnop"
	input := "The key sk-proj-abcdefghijklmnop was used"
	result := RedactSecrets(input, []string{secret})
	if strings.Contains(result, secret) {
		t.Errorf("Full secret was not redacted: %s", result)
	}
}

func TestCollectSecrets(t *testing.T) {
	secrets := map[string]string{
		"api_key":   "sk-123",
		"tenant_id": "tid-456",
		"empty":     "",
	}
	collected := CollectSecrets(secrets)
	if len(collected) != 2 {
		t.Errorf("Expected 2 non-empty secrets, got %d", len(collected))
	}
}

func TestRedactSecrets_NeverLeaksInHealthResponse(t *testing.T) {
	secret := "sk-prod-real-api-key-xyz"
	healthMsg := "Connected to OpenAI with key sk-prod-real-api-key-xyz successfully"
	result := RedactSecrets(healthMsg, []string{secret})
	if strings.Contains(result, secret) {
		t.Fatalf("SECURITY: Secret leaked in health check response: %s", result)
	}
}

func TestRedactSecrets_NeverLeaksInLogOutput(t *testing.T) {
	secret := "anthropic-sk-ant-api-key123"
	logLine := "[INFO] Provider anthropic initialized with key anthropic-sk-ant-api-key123"
	result := RedactSecrets(logLine, []string{secret})
	if strings.Contains(result, secret) {
		t.Fatalf("SECURITY: Secret leaked in log output: %s", result)
	}
}

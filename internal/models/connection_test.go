package models

import (
	"testing"
)

func TestCredential_ToResponse(t *testing.T) {
	cred := &Credential{
		ID:     "123",
		Name:   "My API Key",
		Kind:   ProviderOpenAI,
		Secret: map[string]string{"api_key": "sk-super-secret-key", "org": "my-org"},
	}

	resp := cred.ToResponse()

	if resp.ID != "123" {
		t.Errorf("Expected ID 123, got %s", resp.ID)
	}
	if resp.Name != "My API Key" {
		t.Errorf("Expected Name 'My API Key', got %s", resp.Name)
	}
	if resp.Kind != ProviderOpenAI {
		t.Errorf("Expected Kind %s, got %s", ProviderOpenAI, resp.Kind)
	}

	if val, ok := resp.SecretMasked["api_key"]; !ok || val != "********" {
		t.Errorf("Expected api_key to be masked, got: %v", val)
	}
	if val, ok := resp.SecretMasked["org"]; !ok || val != "********" {
		t.Errorf("Expected org to be masked, got: %v", val)
	}
}

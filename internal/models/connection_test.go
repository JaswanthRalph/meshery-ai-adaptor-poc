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

	if val, ok := resp.HasSecret["api_key"]; !ok || !val {
		t.Error("Expected HasSecret to contain 'api_key' as true")
	}
	if val, ok := resp.HasSecret["org"]; !ok || !val {
		t.Error("Expected HasSecret to contain 'org' as true")
	}
}

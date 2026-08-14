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
	"encoding/json"
	"testing"
)

func TestDesign_Serialization(t *testing.T) {
	design := &Design{
		Name:          "Test Design",
		SchemaVersion: "designs.meshery.io/v1beta1",
		Version:       "1.0.0",
		Components: []Component{
			{
				Name:       "frontend",
				Kind:       "Deployment",
				APIVersion: "apps/v1",
				Model:      "kubernetes",
				Namespace:  "default",
				Config: map[string]interface{}{
					"replicas": 3,
				},
			},
		},
		Relationships: []Relationship{
			{
				Kind:   "edge",
				Type:   "network",
				Source: "frontend",
				Target: "backend",
			},
		},
	}

	bytes, err := json.Marshal(design)
	if err != nil {
		t.Fatalf("Failed to marshal design: %v", err)
	}

	var parsed Design
	if err := json.Unmarshal(bytes, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal design: %v", err)
	}

	if parsed.Name != "Test Design" {
		t.Errorf("Expected name 'Test Design', got '%s'", parsed.Name)
	}
	if len(parsed.Components) != 1 {
		t.Errorf("Expected 1 component, got %d", len(parsed.Components))
	}
	if parsed.Components[0].Name != "frontend" {
		t.Errorf("Expected component name 'frontend', got '%s'", parsed.Components[0].Name)
	}
	if len(parsed.Relationships) != 1 {
		t.Errorf("Expected 1 relationship, got %d", len(parsed.Relationships))
	}
}

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

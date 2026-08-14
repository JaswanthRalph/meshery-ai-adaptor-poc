package ai

import (
	"strings"
	"testing"
)

func TestBuildPromptContext(t *testing.T) {
	userInstruction := "Deploy a simple nginx application"
	input := BuildPromptContext(userInstruction)

	if !strings.Contains(input.UserPrompt, userInstruction) {
		t.Errorf("UserPrompt does not contain the original instruction: %s", input.UserPrompt)
	}

	if !strings.Contains(input.SystemPrompt, "Meshery Design") {
		t.Errorf("SystemPrompt missing 'Meshery Design' context")
	}

	if input.SchemaContext == "" {
		t.Errorf("SchemaContext should not be empty")
	}
}

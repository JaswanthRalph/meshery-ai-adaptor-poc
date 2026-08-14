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

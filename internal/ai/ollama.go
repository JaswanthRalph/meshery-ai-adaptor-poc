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
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

// OllamaProvider implements Provider for local Ollama instances.
// This is the privacy-first option: no data leaves the user's machine.
// No credentials are required.
type OllamaProvider struct {
	conn   *models.Connection
	client *http.Client
}

func NewOllamaProvider(conn *models.Connection, cred *models.Credential) (Provider, error) {
	// Ollama doesn't require credentials — it's local
	return &OllamaProvider{
		conn:   conn,
		client: &http.Client{Timeout: 300 * time.Second}, // Local models can be slow
	}, nil
}

func (p *OllamaProvider) ID() models.ProviderKind { return models.ProviderOllama }
func (p *OllamaProvider) Name() string            { return "Ollama (Local)" }

func (p *OllamaProvider) baseURL() string {
	if url, ok := p.conn.Config["base_url"]; ok && url != "" {
		return url
	}
	return "http://localhost:11434"
}

func (p *OllamaProvider) model() string {
	if m, ok := p.conn.Config["model"]; ok && m != "" {
		return m
	}
	return "llama3.1"
}

func (p *OllamaProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	start := time.Now()
	status := &models.HealthStatus{CheckedAt: time.Now()}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL()+"/api/tags", nil)
	if err != nil {
		status.Status = "error"
		status.Message = err.Error()
		return status, nil
	}

	resp, err := p.client.Do(req)
	status.Latency = time.Since(start).Milliseconds()
	if err != nil {
		status.Status = "unreachable"
		status.Message = fmt.Sprintf("Ollama is not running at %s: %v", p.baseURL(), err)
		return status, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == 200 {
		// Check if the requested model is available
		var result struct {
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		body, _ := io.ReadAll(resp.Body)
		json.Unmarshal(body, &result)

		modelFound := false
		for _, m := range result.Models {
			if m.Name == p.model() || m.Name == p.model()+":latest" {
				modelFound = true
				break
			}
		}

		status.Status = "connected"
		status.ModelInfo = p.model()
		if modelFound {
			status.Message = fmt.Sprintf("Ollama is running, model '%s' is available", p.model())
		} else {
			status.Message = fmt.Sprintf("Ollama is running, but model '%s' may need to be pulled", p.model())
		}
	} else {
		status.Status = "error"
		status.Message = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
	}
	return status, nil
}

func (p *OllamaProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	model := p.model()
	if input.Model != "" {
		model = input.Model
	}

	body := map[string]interface{}{
		"model":  model,
		"prompt": input.SystemPrompt + "\n\n" + input.SchemaContext + "\n\nUser request: " + input.UserPrompt,
		"stream": false,
		"options": map[string]interface{}{
			"temperature": 0.2,
			"num_predict": 4096,
		},
	}

	if input.JSONMode {
		body["format"] = "json"
	}
	if input.TokenStream != nil {
		body["stream"] = true
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL()+"/api/generate", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Ollama request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Ollama returned %d: %s", resp.StatusCode, string(respBody))
	}

	if input.TokenStream != nil {
		defer close(input.TokenStream)
		scanner := bufio.NewScanner(resp.Body)
		var fullContent strings.Builder

		for scanner.Scan() {
			line := scanner.Text()
			var chunk struct {
				Response string `json:"response"`
				Done     bool   `json:"done"`
			}
			if err := json.Unmarshal([]byte(line), &chunk); err == nil {
				if chunk.Response != "" {
					fullContent.WriteString(chunk.Response)
					input.TokenStream <- chunk.Response
				}
				if chunk.Done {
					break
				}
			}
		}

		return &GenerateOutput{
			RawResponse: fullContent.String(),
			Model:       model,
		}, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	var result struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse Ollama response: %w", err)
	}

	return &GenerateOutput{
		RawResponse: result.Response,
		Model:       model,
	}, nil
}

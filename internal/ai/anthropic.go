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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/meshery/ai-adapter-poc/internal/models"
)

// AnthropicProvider implements Provider for the Anthropic Messages API.
type AnthropicProvider struct {
	conn   *models.Connection
	cred   *models.Credential
	client *http.Client
}

func NewAnthropicProvider(conn *models.Connection, cred *models.Credential) (Provider, error) {
	if cred == nil || cred.Secret["api_key"] == "" {
		return nil, fmt.Errorf("anthropic provider requires an api_key credential")
	}
	return &AnthropicProvider{
		conn:   conn,
		cred:   cred,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *AnthropicProvider) ID() models.ProviderKind { return models.ProviderAnthropic }
func (p *AnthropicProvider) Name() string            { return "Anthropic Claude" }

func (p *AnthropicProvider) baseURL() string {
	if url, ok := p.conn.Config["base_url"]; ok && url != "" {
		return url
	}
	return "https://api.anthropic.com"
}

func (p *AnthropicProvider) model() string {
	if m, ok := p.conn.Config["model"]; ok && m != "" {
		return m
	}
	return "claude-sonnet-4-20250514"
}

func (p *AnthropicProvider) anthropicVersion() string {
	if v, ok := p.conn.Config["anthropic_version"]; ok && v != "" {
		return v
	}
	return "2023-06-01"
}

func (p *AnthropicProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	start := time.Now()
	status := &models.HealthStatus{CheckedAt: time.Now()}

	// Anthropic doesn't have a /models endpoint, so we send a minimal request
	body := map[string]interface{}{
		"model":      p.model(),
		"max_tokens": 1,
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL()+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		status.Status = "error"
		status.Message = err.Error()
		return status, nil
	}
	req.Header.Set("x-api-key", p.cred.Secret["api_key"])
	req.Header.Set("anthropic-version", p.anthropicVersion())
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	status.Latency = time.Since(start).Milliseconds()
	if err != nil {
		status.Status = "unreachable"
		status.Message = fmt.Sprintf("Failed to reach Anthropic API: %v", err)
		return status, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		status.Status = "connected"
		status.ModelInfo = p.model()
		status.Message = "Anthropic API is reachable and credentials are valid"
	case 401:
		status.Status = "auth_failed"
		status.Message = "Invalid API key"
	case 429:
		status.Status = "rate_limited"
		status.Message = "Rate limited by Anthropic API"
	default:
		status.Status = "error"
		status.Message = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
	}
	return status, nil
}

func (p *AnthropicProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	model := p.model()
	if input.Model != "" {
		model = input.Model
	}

	body := map[string]interface{}{
		"model":      model,
		"max_tokens": 4096,
		"system":     input.SystemPrompt + "\n\n" + input.SchemaContext,
		"messages":   []map[string]string{{"role": "user", "content": input.UserPrompt}},
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL()+"/v1/messages", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("x-api-key", p.cred.Secret["api_key"])
	req.Header.Set("anthropic-version", p.anthropicVersion())
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("anthropic returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse anthropic response: %w", err)
	}

	if len(result.Content) == 0 {
		return nil, fmt.Errorf("anthropic returned empty content")
	}

	if input.TokenStream != nil {
		input.TokenStream <- result.Content[0].Text
		close(input.TokenStream)
	}

	return &GenerateOutput{
		RawResponse: result.Content[0].Text,
		Model:       model,
		TokensUsed:  result.Usage.InputTokens + result.Usage.OutputTokens,
	}, nil
}

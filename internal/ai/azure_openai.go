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

// AzureOpenAIProvider implements Provider for Azure OpenAI Service.
type AzureOpenAIProvider struct {
	conn   *models.Connection
	cred   *models.Credential
	client *http.Client
}

func NewAzureOpenAIProvider(conn *models.Connection, cred *models.Credential) (Provider, error) {
	if cred == nil || cred.Secret["api_key"] == "" {
		return nil, fmt.Errorf("Azure OpenAI provider requires an api_key credential")
	}
	if conn.Config["resource_name"] == "" || conn.Config["deployment_id"] == "" {
		return nil, fmt.Errorf("Azure OpenAI requires resource_name and deployment_id in config")
	}
	return &AzureOpenAIProvider{
		conn:   conn,
		cred:   cred,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *AzureOpenAIProvider) ID() models.ProviderKind { return models.ProviderAzureOpenAI }
func (p *AzureOpenAIProvider) Name() string            { return "Azure OpenAI" }

func (p *AzureOpenAIProvider) endpoint() string {
	resource := p.conn.Config["resource_name"]
	deployment := p.conn.Config["deployment_id"]
	apiVersion := p.conn.Config["api_version"]
	if apiVersion == "" {
		apiVersion = "2024-02-15-preview"
	}
	return fmt.Sprintf("https://%s.openai.azure.com/openai/deployments/%s/chat/completions?api-version=%s",
		resource, deployment, apiVersion)
}

func (p *AzureOpenAIProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	start := time.Now()
	status := &models.HealthStatus{CheckedAt: time.Now()}

	// Send a minimal completion request to verify connectivity
	body := map[string]interface{}{
		"messages":   []map[string]string{{"role": "user", "content": "ping"}},
		"max_tokens": 1,
	}
	jsonBody, _ := json.Marshal(body)

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint(), bytes.NewReader(jsonBody))
	if err != nil {
		status.Status = "error"
		status.Message = err.Error()
		return status, nil
	}
	req.Header.Set("api-key", p.cred.Secret["api_key"])
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	status.Latency = time.Since(start).Milliseconds()
	if err != nil {
		status.Status = "unreachable"
		status.Message = fmt.Sprintf("Failed to reach Azure OpenAI: %v", err)
		return status, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		status.Status = "connected"
		status.ModelInfo = p.conn.Config["deployment_id"]
		status.Message = "Azure OpenAI is reachable and credentials are valid"
	case 401, 403:
		status.Status = "auth_failed"
		status.Message = "Invalid API key or insufficient permissions"
	case 404:
		status.Status = "error"
		status.Message = "Deployment not found; check resource_name and deployment_id"
	case 429:
		status.Status = "rate_limited"
		status.Message = "Rate limited by Azure OpenAI"
	default:
		status.Status = "error"
		status.Message = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
	}
	return status, nil
}

func (p *AzureOpenAIProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	messages := []map[string]string{
		{"role": "system", "content": input.SystemPrompt + "\n\n" + input.SchemaContext},
		{"role": "user", "content": input.UserPrompt},
	}

	body := map[string]interface{}{
		"messages":    messages,
		"temperature": 0.2,
		"max_tokens":  4096,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint(), bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("api-key", p.cred.Secret["api_key"])
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Azure OpenAI request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Azure OpenAI returned %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage struct {
			TotalTokens int `json:"total_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("Azure OpenAI returned no choices")
	}

	if input.TokenStream != nil {
		input.TokenStream <- result.Choices[0].Message.Content
		close(input.TokenStream)
	}

	return &GenerateOutput{
		RawResponse: result.Choices[0].Message.Content,
		Model:       p.conn.Config["deployment_id"],
		TokensUsed:  result.Usage.TotalTokens,
	}, nil
}

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

// OpenAIProvider implements Provider for the OpenAI API.
type OpenAIProvider struct {
	conn   *models.Connection
	cred   *models.Credential
	client *http.Client
}

// NewOpenAIProvider creates an OpenAI provider from a Connection and Credential.
func NewOpenAIProvider(conn *models.Connection, cred *models.Credential) (Provider, error) {
	if cred == nil || cred.Secret["api_key"] == "" {
		return nil, fmt.Errorf("OpenAI provider requires an api_key credential")
	}
	return &OpenAIProvider{
		conn:   conn,
		cred:   cred,
		client: &http.Client{Timeout: 120 * time.Second},
	}, nil
}

func (p *OpenAIProvider) ID() models.ProviderKind { return models.ProviderOpenAI }
func (p *OpenAIProvider) Name() string            { return "OpenAI" }

func (p *OpenAIProvider) baseURL() string {
	if url, ok := p.conn.Config["base_url"]; ok && url != "" {
		return url
	}
	return "https://api.openai.com/v1"
}

func (p *OpenAIProvider) model() string {
	if m, ok := p.conn.Config["model"]; ok && m != "" {
		return m
	}
	return "gpt-4o"
}

func (p *OpenAIProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	start := time.Now()
	status := &models.HealthStatus{CheckedAt: time.Now()}

	req, err := http.NewRequestWithContext(ctx, "GET", p.baseURL()+"/models", nil)
	if err != nil {
		status.Status = "error"
		status.Message = err.Error()
		return status, nil
	}
	req.Header.Set("Authorization", "Bearer "+p.cred.Secret["api_key"])

	resp, err := p.client.Do(req)
	status.Latency = time.Since(start).Milliseconds()
	if err != nil {
		status.Status = "unreachable"
		status.Message = fmt.Sprintf("Failed to reach OpenAI API: %v", err)
		return status, nil
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case 200:
		status.Status = "connected"
		status.ModelInfo = p.model()
		status.Message = "OpenAI API is reachable and credentials are valid"
	case 401:
		status.Status = "auth_failed"
		status.Message = "Invalid API key"
	case 429:
		status.Status = "rate_limited"
		status.Message = "Rate limited by OpenAI API"
	default:
		status.Status = "error"
		status.Message = fmt.Sprintf("Unexpected status code: %d", resp.StatusCode)
	}
	return status, nil
}

func (p *OpenAIProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	model := p.model()
	if input.Model != "" {
		model = input.Model
	}

	messages := []map[string]string{
		{"role": "system", "content": input.SystemPrompt + "\n\n" + input.SchemaContext},
		{"role": "user", "content": input.UserPrompt},
	}

	body := map[string]interface{}{
		"model":       model,
		"messages":    messages,
		"temperature": 0.2,
		"max_tokens":  4096,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.baseURL()+"/chat/completions", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+p.cred.Secret["api_key"])
	req.Header.Set("Content-Type", "application/json")

	if org, ok := p.conn.Config["organization_id"]; ok && org != "" {
		req.Header.Set("OpenAI-Organization", org)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("OpenAI API request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("OpenAI API returned %d: %s", resp.StatusCode, string(respBody))
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
		return nil, fmt.Errorf("failed to parse OpenAI response: %w", err)
	}

	if len(result.Choices) == 0 {
		return nil, fmt.Errorf("OpenAI returned no choices")
	}

	rawContent := result.Choices[0].Message.Content

	return &GenerateOutput{
		RawResponse: rawContent,
		Model:       model,
		TokensUsed:  result.Usage.TotalTokens,
	}, nil
}

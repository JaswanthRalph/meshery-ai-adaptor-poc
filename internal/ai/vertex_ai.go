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

type VertexAIProvider struct {
	ProjectID   string
	Location    string
	Model       string
	AccessToken string
	BaseURL     string
	client      *http.Client
}

func NewVertexAIProvider(conn *models.Connection, cred *models.Credential) (*VertexAIProvider, error) {
	projectID := conn.Config["project_id"]
	if projectID == "" {
		return nil, fmt.Errorf("project_id is required for Vertex AI")
	}

	location := conn.Config["location"]
	if location == "" {
		location = "us-central1"
	}

	model := conn.Config["model"]
	if model == "" {
		model = "gemini-1.5-pro"
	}

	baseURL := conn.Config["base_url"]

	accessToken := ""
	if cred != nil && cred.Secret != nil {
		accessToken = cred.Secret["access_token"]
	}
	if accessToken == "" {
		return nil, fmt.Errorf("access_token is required for Vertex AI")
	}

	return &VertexAIProvider{
		ProjectID:   projectID,
		Location:    location,
		Model:       model,
		AccessToken: accessToken,
		BaseURL:     baseURL,
		client:      &http.Client{Timeout: 60 * time.Second},
	}, nil
}

func (p *VertexAIProvider) ID() models.ProviderKind {
	return models.ProviderVertexAI
}

func (p *VertexAIProvider) Name() string {
	return "Google Vertex AI"
}

func (p *VertexAIProvider) endpoint(action string) string {
	if p.BaseURL != "" {
		return fmt.Sprintf("%s/v1/projects/%s/locations/%s/publishers/google/models/%s:%s", p.BaseURL, p.ProjectID, p.Location, p.Model, action)
	}
	return fmt.Sprintf("https://%s-aiplatform.googleapis.com/v1/projects/%s/locations/%s/publishers/google/models/%s:%s",
		p.Location, p.ProjectID, p.Location, p.Model, action)
}

func (p *VertexAIProvider) HealthCheck(ctx context.Context) (*models.HealthStatus, error) {
	// A simple way to check auth is to do a lightweight generateContent call with empty contents or countTokens.
	// We'll use countTokens as it's cheaper and faster.
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": "ping"},
				},
			},
		},
	}
	
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint("countTokens"), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	start := time.Now()
	resp, err := p.client.Do(req)
	latency := time.Since(start).Milliseconds()

	if err != nil {
		return &models.HealthStatus{
			Status:    "unreachable",
			Message:   err.Error(),
			Latency:   latency,
			ModelInfo: p.Model,
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return &models.HealthStatus{
			Status:    "auth_failed",
			Message:   fmt.Sprintf("Authentication failed: HTTP %d", resp.StatusCode),
			Latency:   latency,
			ModelInfo: p.Model,
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return &models.HealthStatus{
			Status:    "error",
			Message:   fmt.Sprintf("Vertex AI API error (HTTP %d): %s", resp.StatusCode, string(b)),
			Latency:   latency,
			ModelInfo: p.Model,
		}, nil
	}

	return &models.HealthStatus{
		Status:    "connected",
		Message:   "Successfully connected to Vertex AI",
		Latency:   latency,
		ModelInfo: p.Model,
	}, nil
}

func (p *VertexAIProvider) Generate(ctx context.Context, input *GenerateInput) (*GenerateOutput, error) {
	// Construct Gemini payload
	// System instructions are embedded as part of the user prompt or using systemInstruction field if supported.
	// For simplicity and compatibility, we'll prefix it to the user's prompt.
	fullPrompt := input.SystemPrompt + "\n\nUser Input:\n" + input.UserPrompt

	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"role": "user",
				"parts": []map[string]interface{}{
					{"text": fullPrompt},
				},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.2,
			// No direct json_object response format in some older Gemini APIs unless using tools/function calling, 
			// but we can enforce it via prompt.
		},
	}

	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, "POST", p.endpoint("generateContent"), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.AccessToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("vertex api request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("vertex api returned %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse Vertex response
	var vertexResp struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		UsageMetadata struct {
			TotalTokenCount int `json:"totalTokenCount"`
		} `json:"usageMetadata"`
	}

	if err := json.Unmarshal(respBody, &vertexResp); err != nil {
		return nil, fmt.Errorf("failed to parse vertex response: %w", err)
	}

	if len(vertexResp.Candidates) == 0 || len(vertexResp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("vertex api returned empty response")
	}

	text := vertexResp.Candidates[0].Content.Parts[0].Text

	return &GenerateOutput{
		RawResponse: text,
		Model:       p.Model,
		TokensUsed:  vertexResp.UsageMetadata.TotalTokenCount,
	}, nil
}

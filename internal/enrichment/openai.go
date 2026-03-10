package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

const defaultOpenAIModel = "qwen2.5-coder:7b"

// openaiProvider implements provider using OpenAI-compatible chat completions API.
type openaiProvider struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
	name    string
}

func newOpenAIProvider(baseURL, providerName string, timeout time.Duration) *openaiProvider {
	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = defaultOpenAIModel
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		apiKey = "local-llm"
	}
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	return &openaiProvider{
		apiKey:  apiKey,
		baseURL: baseURL,
		model:   model,
		client:  &http.Client{Timeout: timeout},
		name:    providerName,
	}
}

func (p *openaiProvider) Name() string { return p.name + ":" + p.model }

type openaiRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	Messages  []openaiMessage `json:"messages"`
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *openaiProvider) Complete(ctx context.Context, prompt string) (string, error) {
	reqBody := openaiRequest{
		Model:     p.model,
		MaxTokens: 150,
		Messages:  []openaiMessage{{Role: "user", Content: prompt}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var or openaiResponse
	if err := json.Unmarshal(respBody, &or); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if or.Error != nil {
		return "", fmt.Errorf("api error: %s", or.Error.Message)
	}
	if len(or.Choices) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return or.Choices[0].Message.Content, nil
}

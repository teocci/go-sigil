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

const (
	defaultGoogleModel = "gemini-2.0-flash"
	googleBaseURL      = "https://generativelanguage.googleapis.com"
)

// googleProvider implements provider using Google Gemini REST API.
type googleProvider struct {
	apiKey string
	model  string
	client *http.Client
}

func newGoogleProvider(timeout time.Duration) *googleProvider {
	model := os.Getenv("GOOGLE_MODEL")
	if model == "" {
		model = defaultGoogleModel
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &googleProvider{
		apiKey: os.Getenv("GOOGLE_API_KEY"),
		model:  model,
		client: &http.Client{Timeout: timeout},
	}
}

func (p *googleProvider) Name() string { return "google:" + p.model }

type googleRequest struct {
	Contents []googleContent `json:"contents"`
}

type googleContent struct {
	Parts []googlePart `json:"parts"`
}

type googlePart struct {
	Text string `json:"text"`
}

type googleResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (p *googleProvider) Complete(ctx context.Context, prompt string) (string, error) {
	reqBody := googleRequest{
		Contents: []googleContent{{Parts: []googlePart{{Text: prompt}}}},
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/v1beta/models/%s:generateContent?key=%s", googleBaseURL, p.model, p.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	var gr googleResponse
	if err := json.Unmarshal(respBody, &gr); err != nil {
		return "", fmt.Errorf("unmarshal response: %w", err)
	}
	if gr.Error != nil {
		return "", fmt.Errorf("api error: %s", gr.Error.Message)
	}
	if len(gr.Candidates) == 0 || len(gr.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response")
	}
	return gr.Candidates[0].Content.Parts[0].Text, nil
}

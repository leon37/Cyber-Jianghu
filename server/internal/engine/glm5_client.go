package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sashabaranov/go-openai"
)

const (
	defaultBaseURL = "https://open.bigmodel.cn/api/paas/v4"
	defaultTimeout = 120 * time.Second
	maxRetries     = 3
	retryDelay     = 1 * time.Second
)

// GLM5Client wraps the OpenAI client for GLM-5 API compatibility
type GLM5Client struct {
	client     *openai.Client
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// ChatMessage represents a chat message
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Messages    []ChatMessage `json:"messages"`
	Model       string        `json:"model"`
	Temperature float64       `json:"temperature,omitempty"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`
	Error   *APIError `json:"error,omitempty"`
}

// Choice represents a response choice
type Choice struct {
	Index        int         `json:"index"`
	Message      ChatMessage `json:"message"`
	FinishReason string      `json:"finish_reason"`
}

// Usage represents token usage
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// APIError represents an API error
type APIError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
	Code    string `json:"code"`
}

// EmbeddingRequest represents an embedding request
type EmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

// EmbeddingResponse represents an embedding response
type EmbeddingResponse struct {
	Object string       `json:"object"`
	Data   []Embedding  `json:"data"`
	Model  string       `json:"model"`
	Usage  EmbeddingUsage `json:"usage"`
	Error  *APIError    `json:"error,omitempty"`
}

// Embedding represents an embedding vector
type Embedding struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// EmbeddingUsage represents embedding usage
type EmbeddingUsage struct {
	PromptTokens int `json:"prompt_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NewGLM5Client creates a new GLM-5 API client
func NewGLM5Client(apiKey string) *GLM5Client {
	config := openai.DefaultConfig(apiKey)
	config.BaseURL = defaultBaseURL

	httpClient := &http.Client{
		Timeout: defaultTimeout,
	}

	return &GLM5Client{
		client:     openai.NewClientWithConfig(config),
		baseURL:    defaultBaseURL,
		apiKey:     apiKey,
		httpClient: httpClient,
	}
}

// Chat sends a chat completion request
func (c *GLM5Client) Chat(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	var lastErr error

	for attempt := 0; attempt < maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryDelay * time.Duration(attempt)):
			}
		}

		response, err := c.doChatRequest(ctx, req)
		if err == nil {
			return response, nil
		}

		lastErr = err
		if !isRetryableError(err) {
			break
		}
	}

	return nil, fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

// doChatRequest performs the actual HTTP request
func (c *GLM5Client) doChatRequest(ctx context.Context, req *ChatRequest) (*ChatResponse, error) {
	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/chat/completions", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error *APIError `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		if errorResp.Error != nil {
			return nil, fmt.Errorf("API error: %s (code: %s)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var chatResp ChatResponse
	if err := json.Unmarshal(respBody, &chatResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if chatResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	return &chatResp, nil
}

// CreateEmbedding creates embeddings for the given texts
func (c *GLM5Client) CreateEmbedding(ctx context.Context, texts []string, model string) (*EmbeddingResponse, error) {
	req := &EmbeddingRequest{
		Input: texts,
		Model: model,
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/embeddings", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errorResp struct {
			Error *APIError `json:"error"`
		}
		if err := json.Unmarshal(respBody, &errorResp); err != nil {
			return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
		}
		if errorResp.Error != nil {
			return nil, fmt.Errorf("API error: %s (code: %s)", errorResp.Error.Message, errorResp.Error.Code)
		}
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	var embResp EmbeddingResponse
	if err := json.Unmarshal(respBody, &embResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if embResp.Error != nil {
		return nil, fmt.Errorf("API error: %s", embResp.Error.Message)
	}

	return &embResp, nil
}

// isRetryableError checks if an error is retryable
func isRetryableError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return contains(errStr, "timeout") ||
		contains(errStr, "connection refused") ||
		contains(errStr, "rate limit")
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		len(substr) == 0 ||
		stringsContains(s, substr))
}

func stringsContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

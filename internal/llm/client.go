package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/bprendie/weazlwrite/internal/config"
)

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content,omitempty"`
}

type Client struct {
	provider config.Provider
	http     *http.Client
}

func New(provider config.Provider) Client {
	return Client{
		provider: provider,
		http:     &http.Client{Timeout: 2 * time.Minute},
	}
}

func (c Client) GenerateBlock(ctx context.Context, document, instruction string) (string, error) {
	messages := []ChatMessage{
		{
			Role: "system",
			Content: "You are an inline Markdown writing assistant for a terminal Markdown editor. " +
				"Generate only the block to insert at the cursor. Do not include commentary, apologies, surrounding explanations, or markdown fences unless the user asks for code or a fenced block is the right Markdown formatting. " +
				"Make the result ready to paste into the document.",
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Current document:\n\n%s\n\nInstruction for the block to insert:\n%s", trimDocument(document), instruction),
		},
	}
	switch strings.ToLower(c.provider.Type) {
	case "vllm":
		return c.completeOpenAICompat(ctx, messages, 900)
	case "ollama":
		return c.completeOllama(ctx, messages, 900)
	default:
		return "", fmt.Errorf("unsupported provider type %q", c.provider.Type)
	}
}

func (c Client) completeOpenAICompat(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	reqBody := map[string]any{
		"model":       c.provider.Model,
		"messages":    messages,
		"temperature": 0.2,
		"stream":      false,
		"max_tokens":  maxTokens,
	}
	resp, err := c.post(ctx, "/v1/chat/completions", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		Choices []struct {
			Message ChatMessage `json:"message"`
		} `json:"choices"`
		Error *struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Error != nil {
		return "", errors.New(body.Error.Message)
	}
	if len(body.Choices) == 0 {
		return "", errors.New("empty completion response")
	}
	return strings.TrimSpace(body.Choices[0].Message.Content), nil
}

func (c Client) completeOllama(ctx context.Context, messages []ChatMessage, maxTokens int) (string, error) {
	reqBody := map[string]any{
		"model":    c.provider.Model,
		"messages": messages,
		"stream":   false,
		"options": map[string]any{
			"num_predict": maxTokens,
			"temperature": 0.2,
		},
	}
	resp, err := c.post(ctx, "/api/chat", reqBody)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var body struct {
		Message ChatMessage `json:"message"`
		Error   string      `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return "", err
	}
	if body.Error != "" {
		return "", errors.New(body.Error)
	}
	return strings.TrimSpace(body.Message.Content), nil
}

func (c Client) post(ctx context.Context, path string, body any) (*http.Response, error) {
	b, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	url := baseURL(c.provider) + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.provider.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.provider.APIKey)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		detail := strings.TrimSpace(string(body))
		if detail == "" {
			return nil, fmt.Errorf("%s returned %s", url, resp.Status)
		}
		return nil, fmt.Errorf("%s returned %s: %s", url, resp.Status, detail)
	}
	return resp, nil
}

func baseURL(provider config.Provider) string {
	u := strings.TrimRight(strings.TrimSpace(provider.ServerURL), "/")
	switch strings.ToLower(provider.Type) {
	case "vllm":
		u = strings.TrimSuffix(u, "/v1")
	case "ollama":
		u = strings.TrimSuffix(u, "/api")
	}
	return u
}

func trimDocument(document string) string {
	const maxRunes = 12000
	runes := []rune(document)
	if len(runes) <= maxRunes {
		return document
	}
	return string(runes[len(runes)-maxRunes:])
}

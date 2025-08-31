package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"
)

type Client struct {
	baseURL string
	log     *slog.Logger
	client  *http.Client
}

type TagModel struct {
	Name       string    `json:"name"`
	Model      string    `json:"model"`
	Digest     string    `json:"digest"`
	Size       int64     `json:"size"`
	ModifiedAt time.Time `json:"modified_at"`
	Details    any       `json:"details"`
}

func NewClient(baseURL string, log *slog.Logger) *Client {
	return &Client{
		baseURL: baseURL,
		log:     log,
		client:  &http.Client{Timeout: 240 * time.Second}, // to handle log ops like model pull
	}
}

func (c *Client) Ping(ctx context.Context) error {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/version", c.baseURL), nil)
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	data, _ := io.ReadAll(res.Body)
	res.Body.Close()
	c.log.Info("ping response", "response", string(data))
	if res.StatusCode > 400 {
		return fmt.Errorf("ollama ping status: %d", res.StatusCode)
	}

	return nil
}

// Generate sends a single-turn generation (non-stream) via /api/generate.
func (c *Client) Generate(ctx context.Context, model, prompt string) (string, time.Duration, error) {
	payload := map[string]any{"model": model, "prompt": prompt, "stream": false}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/generate", c.baseURL), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	start := time.Now()
	res, err := c.client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		body, _ := io.ReadAll(res.Body)
		return "", 0, fmt.Errorf("ollama generate: %s", string(body))
	}
	var out struct {
		Response string `json:"response"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return "", 0, err
	}
	return out.Response, time.Since(start), nil
}

// Tags lists local models via GET /api/tags.
func (c *Client) Tags(ctx context.Context) ([]TagModel, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("%s/api/tags", c.baseURL), nil)
	res, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("ollama tags: %s", res.Status)
	}
	var out struct {
		Models []TagModel `json:"models"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		return nil, err
	}
	return out.Models, nil
}

// create a second client with no timeout for long ops:
var httpNoTimeout = &http.Client{Timeout: 0}

// Pull downloads a model locally via POST /api/pull.
func (c *Client) Pull(ctx context.Context, name string) error {
	if name == "" {
		return errors.New("empty model name")
	}
	payload := map[string]any{"name": name, "stream": false}
	b, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("%s/api/pull", c.baseURL), bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}

	body, _ := io.ReadAll(res.Body)
	c.log.Info("ollama pull response", "res", string(body))

	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return fmt.Errorf("ollama pull: %s", string(body))
	}
	return nil
}

func trimSlash(s string) string {
	if len(s) > 0 && s[len(s)-1] == '/' {
		return s[:len(s)-1]
	}
	return s
}

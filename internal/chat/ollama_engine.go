package chat

import (
	"context"
	"github.com/varsilias/zero-downtime/internal/ollama"
	"time"
)

type OllamaEngine struct {
	c *ollama.Client
}

func NewOllamaEngine(c *ollama.Client) *OllamaEngine {
	return &OllamaEngine{
		c: c,
	}
}

func (e *OllamaEngine) Generate(ctx context.Context, model, prompt string) (string, time.Duration, error) {
	return e.c.Generate(ctx, model, prompt)
}

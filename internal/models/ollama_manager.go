package models

import (
	"context"
	"github.com/varsilias/zero-downtime/internal/ollama"
)

type OllamaManager struct{ c *ollama.Client }

func NewOllamaManager(c *ollama.Client) *OllamaManager { return &OllamaManager{c: c} }

func (m *OllamaManager) List(ctx context.Context) ([]string, error) {
	items, err := m.c.Tags(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, it := range items {
		out = append(out, it.Name)
	}
	return out, nil
}

func (m *OllamaManager) Healthy(ctx context.Context, model string) error {
	// best-effort: if it’s in tags, we consider it healthy
	items, err := m.c.Tags(ctx)
	if err != nil {
		return err
	}
	for _, it := range items {
		if it.Name == model {
			return nil
		}
	}
	return ErrUnknownModel
}

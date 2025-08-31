package models

import (
	"context"
	"errors"
)

type Manager interface {
	List(ctx context.Context) ([]string, error)
	Healthy(ctx context.Context, model string) error
}

type StaticManager struct{ items []string }

func NewStaticManager(items []string) *StaticManager { return &StaticManager{items: items} }

func (m *StaticManager) List(ctx context.Context) ([]string, error) {
	return append([]string(nil), m.items...), nil
}

func (m *StaticManager) Healthy(ctx context.Context, model string) error {
	for _, x := range m.items {
		if x == model {
			return nil
		}
	}
	return errors.New("unknown model")
}

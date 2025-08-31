package chat

import (
	"context"
	"fmt"
	"time"
)

type Engine interface {
	Generate(ctx context.Context, model, prompt string) (text string, latency time.Duration, err error)
}

type EchoEngine struct {
	minLatency time.Duration
}

func NewEchoEngine(minLatency time.Duration) *EchoEngine { return &EchoEngine{minLatency: minLatency} }

func (e *EchoEngine) Generate(ctx context.Context, model, prompt string) (string, time.Duration, error) {
	start := time.Now()
	if e.minLatency > 0 {
		time.Sleep(e.minLatency)
	}
	text := fmt.Sprintf("(demo:%s) you said: %s", model, prompt)
	return text, time.Since(start), nil
}

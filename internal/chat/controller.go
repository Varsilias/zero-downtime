package chat

import (
	"context"
	"log/slog"
	"time"

	"github.com/varsilias/zero-downtime/internal/session"
	"github.com/varsilias/zero-downtime/pkg/types"
)

type Controller struct {
	log      *slog.Logger
	eng      Engine
	sessions session.Store
}

func NewController(log *slog.Logger, eng Engine, store session.Store) *Controller {
	return &Controller{log: log, eng: eng, sessions: store}
}

// Chat orchestrates a single turn: persist user msg, call engine, persist assistant reply.
func (c *Controller) Chat(ctx context.Context, sessionID, model, prompt string) (types.Message, time.Duration, error) {
	c.log.Info("chat", "calling engine with model", model)
	user := types.Message{Role: types.RoleUser, Content: prompt, Timestamp: time.Now()}
	if err := c.sessions.Append(sessionID, user); err != nil {
		return types.Message{}, 0, err
	}

	text, latency, err := c.eng.Generate(ctx, model, prompt)
	if err != nil {
		c.log.Error("engine call", "error from engine server", err.Error())
		return types.Message{}, 0, err
	}
	assistant := types.Message{Role: types.RoleAssistant, Content: text, Timestamp: time.Now()}
	if err := c.sessions.Append(sessionID, assistant); err != nil {
		return types.Message{}, 0, err
	}
	return assistant, latency, nil
}

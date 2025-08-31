package api

import (
	"encoding/json"
	"github.com/varsilias/zero-downtime/internal/buildinfo"
	"github.com/varsilias/zero-downtime/internal/chat"
	"github.com/varsilias/zero-downtime/internal/models"
	"github.com/varsilias/zero-downtime/internal/session"
	"github.com/varsilias/zero-downtime/pkg/utils"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type Handlers struct {
	log      *slog.Logger
	chat     *chat.Controller
	models   models.Manager
	sessions session.Store
	Admin    *Admin
}

func NewHandlers(log *slog.Logger, chatCtrl *chat.Controller, manager models.Manager, store session.Store) *Handlers {
	return &Handlers{
		log:      log,
		chat:     chatCtrl,
		models:   manager,
		sessions: store,
	}
}

// Health is a basic liveness endpoint.
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{
		"status":    true,
		"message":   "zero-downtime-demo",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handlers) Version(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{
		"version":  buildinfo.Version,
		"commit":   buildinfo.Commit,
		"built_at": buildinfo.BuiltAt,
	}

	utils.JSON(w, http.StatusOK, res)
}

// ListModels GET /api/models
func (h *Handlers) ListModels(w http.ResponseWriter, r *http.Request) {
	res := map[string]any{
		"models": []string{"llama2", "mistral", "phi3"},
	}
	// TODO: query Model Manager / Ollama runtime
	utils.JSON(w, http.StatusOK, res)
}

func (h *Handlers) Chat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Model     string `json:"model"`
		Message   string `json:"message"`
		SessionID string `json:"session_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, http.StatusBadRequest, map[string]any{"error": "invalid json"})
		return
	}
	if req.Model == "" || req.Message == "" {
		utils.JSON(w, http.StatusBadRequest, map[string]any{"error": "model and message are required"})
		return
	}
	if req.SessionID == "" {
		req.SessionID = "default"
	}

	msg, latency, err := h.chat.Chat(r.Context(), req.SessionID, req.Model, req.Message)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	utils.JSON(w, http.StatusOK, map[string]any{
		"response":   msg.Content,
		"timestamp":  msg.Timestamp.UTC().Format(time.RFC3339),
		"latency_ms": latency.Milliseconds(),
		"model":      req.Model,
		"session_id": req.SessionID,
	})
}

// GetHistory GET /api/history/:session_id
func (h *Handlers) GetHistory(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	const prefix = "/api/history/"
	if !strings.HasPrefix(path, prefix) {
		http.NotFound(w, r)
		return
	}
	sessionID := strings.TrimPrefix(path, prefix)
	if sessionID == "" {
		utils.JSON(w, http.StatusBadRequest, map[string]any{"error": "missing session_id"})
		return
	}

	history, err := h.sessions.Get(sessionID)
	if err != nil {
		utils.JSON(w, http.StatusInternalServerError, map[string]any{"error": err.Error()})
		return
	}

	// Shape history for the contract
	out := make([]map[string]string, 0, len(history))
	for _, m := range history {
		out = append(out, map[string]string{"role": string(m.Role), "content": m.Content})
	}
	utils.JSON(w, http.StatusOK, map[string]any{"history": out})
}

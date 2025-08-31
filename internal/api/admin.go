package api

import (
	"encoding/json"
	"github.com/varsilias/zero-downtime/internal/ollama"
	"github.com/varsilias/zero-downtime/pkg/utils"
	"net/http"
)

type Admin struct{ oc *ollama.Client }

func NewAdmin(oc *ollama.Client) *Admin { return &Admin{oc: oc} }

// PullModel POST /admin/models/pull { name }
func (a *Admin) PullModel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.JSON(w, 400, map[string]any{"error": "invalid json"})
		return
	}
	if req.Name == "" {
		utils.JSON(w, 400, map[string]any{"error": "name required"})
		return
	}
	if err := a.oc.Pull(r.Context(), req.Name); err != nil {
		utils.JSON(w, 500, map[string]any{"error": err.Error()})
		return
	}
	utils.JSON(w, 200, map[string]any{"ok": true})
}

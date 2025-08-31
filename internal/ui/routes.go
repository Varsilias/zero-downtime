package ui

import (
	"github.com/go-chi/chi/v5"
	"github.com/varsilias/zero-downtime/internal/buildinfo"
	"github.com/varsilias/zero-downtime/internal/session"
	"net/http"
	"sort"
	"strings"
	"time"
)

type HomeData struct {
	Models    []string
	SessionID string
	History   []struct{ Role, Content string }
}

func RegisterRoutes(mux *chi.Mux, h *UI) {
	mux.Get("/", h.Home)
	mux.Post("/ui/chat", h.ChatPost)
	mux.Post("/ui/session/new", h.NewSession)
	mux.Get("/ui/version-pill", h.VersionPill)
}

// Home shows the chat UI. Optional session via query: /?s=<id>
func (u *UI) Home(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(r.URL.Query().Get("s"))
	if sid == "" {
		sid = "default"
	}

	// preload models
	mods, _ := u.models.List(r.Context())

	// history
	msgs, _ := u.sessions.Get(sid)
	hist := make([]MsgView, 0, len(msgs))
	for _, m := range msgs {
		hist = append(hist, MsgView{Role: string(m.Role), HTML: u.mdHTML(m.Content)})
	}

	// sessions list (best effort if memory store)
	var sessions []session.Summary
	if mem, ok := u.sessions.(*session.MemoryStore); ok {
		sessions = mem.List()
		sort.SliceStable(sessions, func(i, j int) bool { return sessions[i].Updated.After(sessions[j].Updated) })
	}

	u.render(w, "chat.html", map[string]any{
		"Models":    mods,
		"SessionID": sid,
		"History":   hist,
		"Sessions":  sessions,
		"Commit":    buildinfo.Commit,
		"Version":   buildinfo.Version,
		"BuiltAt":   buildinfo.BuiltAt,
	}, http.StatusOK)
}

// ChatPost returns *two fragments*: user bubble then assistant bubble.
func (u *UI) ChatPost(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	model := r.Form.Get("model")
	msg := strings.TrimSpace(r.Form.Get("message"))
	sid := r.Form.Get("session_id")
	if sid == "" {
		sid = "default"
	}
	if model == "" || msg == "" {
		http.Error(w, "bad request", 400)
		return
	}

	// Optimistically render user bubble first
	user := MsgView{Role: "user", HTML: u.mdHTML(msg)}
	if err := u.tpl.ExecuteTemplate(w, "message.html", user); err != nil {
		u.errTpl(w, err)
		return
	}

	// Then compute assistant reply via controller
	reply, latency, err := u.chat.Chat(r.Context(), sid, model, msg)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	assistant := MsgView{Role: "assistant", HTML: u.mdHTML(reply.Content), Latency: latency.Milliseconds(), At: time.Now().Format(time.RFC822)}
	_ = u.tpl.ExecuteTemplate(w, "message.html", assistant)
}

// NewSession creates a fresh session ID and redirects to /?s=...
func (u *UI) NewSession(w http.ResponseWriter, r *http.Request) {
	id := newID()
	if mem, ok := u.sessions.(*session.MemoryStore); ok {
		mem.Touch(id)
	}
	url := "/?s=" + id

	// If this is an HTMX request, instruct client to redirect
	if r.Header.Get("HX-Request") == "true" {
		w.Header().Set("HX-Redirect", url)
		w.WriteHeader(http.StatusNoContent) // 204 + HX-Redirect -> full navigation
		return
	}

	// Fallback for non-HTMX requests
	http.Redirect(w, r, url, http.StatusFound)
}

type versionVM struct {
	Version string
	Commit  string
	BuiltAt string
}

func (u *UI) VersionPill(w http.ResponseWriter, r *http.Request) {
	// Fragment response; avoid caching so rollouts show quickly
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	data := versionVM{
		Version: buildinfo.Version,
		Commit:  buildinfo.Commit,
		BuiltAt: buildinfo.BuiltAt,
	}
	if err := u.tpl.ExecuteTemplate(w, "version-pill.html", data); err != nil {
		u.errTpl(w, err)
	}
}

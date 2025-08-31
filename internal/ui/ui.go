package ui

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/microcosm-cc/bluemonday"
	"github.com/varsilias/zero-downtime/internal/chat"
	"github.com/varsilias/zero-downtime/internal/models"
	"github.com/varsilias/zero-downtime/internal/session"
	"github.com/yuin/goldmark"
	highlighting "github.com/yuin/goldmark-highlighting/v2"
	gmhtml "github.com/yuin/goldmark/renderer/html"
	"html/template"
	"log/slog"
	"net/http"
	"time"
)

type UI struct {
	log      *slog.Logger
	tpl      *template.Template
	chat     *chat.Controller
	models   models.Manager
	sessions session.Store
	md       goldmark.Markdown
}

func New(log *slog.Logger, c *chat.Controller, m models.Manager, s session.Store) (*UI, error) {
	// Parse all templates (layout + pages + partials)
	// Keeping it simple with disk parsing for now
	t := template.New("root")
	var err error
	if t, err = t.ParseGlob("web/templates/*.html"); err != nil {
		return nil, err
	}
	if t, err = t.ParseGlob("web/templates/partials/*.html"); err != nil {
		return nil, err
	}

	md := goldmark.New(
		goldmark.WithRendererOptions(gmhtml.WithUnsafe()),
		goldmark.WithExtensions(
			highlighting.NewHighlighting(
				highlighting.WithStyle("dracula"),
				highlighting.WithFormatOptions(
					// Use inline styles so we don’t need an external CSS file
					chromahtml.WithLineNumbers(false),
				),
			),
		),
	)

	return &UI{
		log:      log,
		tpl:      t,
		chat:     c,
		models:   m,
		sessions: s,
		md:       md,
	}, nil
}

type MsgView struct {
	Role    string
	HTML    template.HTML
	Latency int64
	At      string
}

func (u *UI) mdHTML(src string) template.HTML {
	var buf bytes.Buffer
	_ = u.md.Convert([]byte(src), &buf)

	p := bluemonday.UGCPolicy()
	p.AllowAttrs("class").OnElements("code", "pre", "span")
	p.AllowAttrs("style").OnElements("span") // enable inline styles from highlighter

	safe := p.SanitizeBytes(buf.Bytes())
	return template.HTML(safe)
}

func (u *UI) render(w http.ResponseWriter, name string, data any, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := u.tpl.ExecuteTemplate(w, name, data); err != nil {
		u.errTpl(w, err)
	}
}

func (u *UI) errTpl(w http.ResponseWriter, err error) {
	u.log.Error("template execute", "err", err)
	_, _ = w.Write([]byte("<pre>template error: " + err.Error() + "</pre>"))
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return time.Now().Format("20060102150405")
	}
	return hex.EncodeToString(b[:])
}

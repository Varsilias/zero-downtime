package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/varsilias/zero-downtime/internal/api"
	"github.com/varsilias/zero-downtime/internal/buildinfo"
	"github.com/varsilias/zero-downtime/internal/chat"
	"github.com/varsilias/zero-downtime/internal/logging"
	"github.com/varsilias/zero-downtime/internal/middleware"
	"github.com/varsilias/zero-downtime/internal/models"
	"github.com/varsilias/zero-downtime/internal/ollama"
	"github.com/varsilias/zero-downtime/internal/session"
	"github.com/varsilias/zero-downtime/internal/ui"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	addr := flag.String("addr", getEnv("ADDR", "8080"), "HTTP listen address")
	level := flag.String("log-level", getEnv("LOG_LEVEL", "info"), "log level: debug|info|warn|error")
	json := flag.Bool("log-json", getEnv("LOG_JSON", "false") == "true", "log as JSON")
	ollamaURL := flag.String("ollama", getEnv("OLLAMA_BASE_URL", "http://localhost:11434"), "Ollama base URL")

	// ollama read knobs
	waitEnabled := strings.ToLower(getEnv("OLLAMA_WAIT", "true")) == "true"
	waitTimeout, _ := time.ParseDuration(getEnv("OLLAMA_WAIT_TIMEOUT", "180s"))
	waitInterval, _ := time.ParseDuration(getEnv("OLLAMA_WAIT_INTERVAL", "2s"))
	waitModels := strings.Fields(getEnv("OLLAMA_WAIT_MODELS", "gemma3:270m smollm:135m deepseek-r1:1.5b")) // "llama3.2 mistral"

	flag.Parse()

	ollamaActive := false

	logger := logging.New(*level, *json)
	logger.Info("build", "version", buildinfo.Version, "commit", buildinfo.Commit, "built_at", buildinfo.BuiltAt)
	logger.Info("Lord speak you server is listening", "port", *addr, "ollama", *ollamaURL)

	// Dependencies (prefer Ollama if reachable; else fall back to echo)
	var (
		engine    chat.Engine
		modelsMgr models.Manager
	)

	oc := ollama.NewClient(*ollamaURL, logger)
	if waitEnabled {
		logger.Info("waiting for Ollama", "timeout", waitTimeout.String(), "interval", waitInterval.String(), "models", waitModels)
		ctxWait, cancel := context.WithTimeout(context.Background(), waitTimeout)
		err := waitForOllama(ctxWait, oc, waitModels, waitInterval, logger)
		cancel()
		if err != nil {
			logger.Warn("Ollama wait timed out; continuing with fallback", "err", err.Error())
		} else {
			logger.Info("Ollama is ready (API + required models present)")
		}
	}

	if err := oc.Ping(context.Background()); err == nil {
		logger.Info("ollama reachable: enabling ollama engine")
		engine = chat.NewOllamaEngine(oc)
		modelsMgr = models.NewOllamaManager(oc)
		ollamaActive = true
	} else {
		logger.Warn("ollama not reachable; falling back to echo engine", "err", err)
		modelsMgr = models.NewStaticManager([]string{"llama2", "mistral", "phi3"})
		engine = chat.NewEchoEngine(30 * time.Millisecond)
	}

	sessionStore := session.NewMemoryStore()
	chatCtrl := chat.NewController(logger, engine, sessionStore)

	uih, err := ui.New(logger, chatCtrl, modelsMgr, sessionStore)
	if err != nil {
		logger.Error("ui init", "err", err)
		os.Exit(1)
	}

	h := api.NewHandlers(logger, chatCtrl, modelsMgr, sessionStore)
	if ollamaActive {
		h.Admin = api.NewAdmin(oc)
	}
	mux := chi.NewRouter()

	mux.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("web/static"))))

	ui.RegisterRoutes(mux, uih)
	api.RegisterRoutes(mux, h)

	var handler http.Handler = mux
	handler = middleware.Recoverer(logger)(handler)
	handler = middleware.RequestID()(handler)
	handler = middleware.AccessLog(logger)(handler)
	handler = middleware.VersionHeader(logger)(handler)

	server := http.Server{
		Addr:              fmt.Sprintf(":%s", *addr),
		Handler:           handler,
		ReadTimeout:       15 * time.Second,
		ReadHeaderTimeout: 15 * time.Second,
		//WriteTimeout:      60 * time.Second,
		WriteTimeout: 5 * time.Minute, // using this to allow more time for request that take much time like pulling a model from ollama registry
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	errChan := make(chan error, 1)
	go func() { errChan <- server.ListenAndServe() }()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		if errors.Is(err, http.ErrServerClosed) {
			logger.Error("server error", "err", err)
			os.Exit(1)
		}
	case sig := <-sigChan:
		logger.Info("shutdown signal received", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		logger.Error("graceful shutdown failed", "err", err)
	} else {
		logger.Info("server stopped")
	}
}

func waitForOllama(ctx context.Context, oc *ollama.Client, models []string, interval time.Duration, log *slog.Logger) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	check := func() error {
		if err := oc.Ping(ctx); err != nil {
			return fmt.Errorf("ollama not reachable: %w", err)
		}

		if len(models) == 0 {
			return nil // only API readiness required
		}

		tags, err := oc.Tags(ctx)
		if err != nil {
			return fmt.Errorf("list tags: %w", err)
		}

		have := map[string]struct{}{}
		for _, t := range tags {
			have[t.Name] = struct{}{}
		}

		for _, m := range models {
			if _, ok := have[m]; !ok {
				return fmt.Errorf("model not present yet: %s", m)
			}
		}

		return nil
	}

	// do an immediate attempt first
	if err := check(); err == nil {
		return nil
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := check(); err == nil {
				return nil
			}
		}
	}
}

func getEnv(key, def string) string {
	v := os.Getenv(key)
	if v != "" {
		return v
	}
	return def
}

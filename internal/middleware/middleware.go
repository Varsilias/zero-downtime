package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"github.com/varsilias/zero-downtime/internal/buildinfo"
	"io"
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"
	"time"
)

// RequestIDKey is the context key for the request ID
type RequestIDKey struct{}

func RequestID() func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := newID()
			ctx := context.WithValue(r.Context(), RequestIDKey{}, id)
			w.Header().Set("X-Request-ID", id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func newID() string {
	var b [8]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		return "0000000000000000"
	}

	return hex.EncodeToString(b[:])
}

// Recoverer converts panics to 500s and logs the stack.
func Recoverer(logger *slog.Logger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic",
						"err", rec,
						"stack", string(debug.Stack()),
					)
					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// statusWriter captures the HTTP status for access logs.
type statusWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(p []byte) (int, error) {
	if w.status == 0 {
		w.status = 200
	}
	n, err := w.ResponseWriter.Write(p)
	w.size += n
	return n, err
}

// AccessLog writes concise request logs using slog.
func AccessLog(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &statusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			dur := time.Since(start)

			// Pull request ID if present
			var reqID string
			if v := r.Context().Value(RequestIDKey{}); v != nil {
				reqID = v.(string)
			}

			logger.Info(
				"http",
				"method", r.Method,
				"path", r.URL.Path,
				"status", sw.status,
				"bytes", sw.size,
				"remote", remoteIP(r.RemoteAddr),
				"duration_ms", dur.Milliseconds(),
				"req_id", reqID,
			)
		})
	}
}

func VersionHeader(logger *slog.Logger) func(handler http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-App-Version", buildinfo.Version)
			w.Header().Set("X-App-Commit", buildinfo.Commit)
			w.Header().Set("X-App-Built-At", buildinfo.BuiltAt)
			next.ServeHTTP(w, r)
		})
	}
}

func remoteIP(remoteAddr string) string {
	// remoteAddr is usually ip:port; keep ip part for readability
	if i := strings.LastIndex(remoteAddr, ":"); i > 0 {
		return remoteAddr[:i]
	}
	return remoteAddr
}

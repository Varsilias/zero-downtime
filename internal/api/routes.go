package api

import "github.com/go-chi/chi/v5"

func RegisterRoutes(mux *chi.Mux, h *Handlers) {
	mux.Get("/healthz", h.Health)
	mux.Get("/version", h.Version)

	mux.Post("/api/chat", h.Chat)
	mux.Get("/api/models", h.ListModels)

	mux.Get("/api/history/", h.GetHistory)
	if h.Admin != nil {
		mux.Post("/admin/models/pull", h.Admin.PullModel)
	}
}

# Zero-Downtime AI Chat (Go + HTMX + Tailwind + Ollama)

A compact, production-style demo that feels like a real LLM chat app and doubles as a **zero-downtime deployment** showcase.

- **Go** backend with clean layers (API, chat controller, model manager, sessions, logging)
- **HTMX + Tailwind** UI (server-rendered; no SPA framework)
- **Ollama** integration (local or Kubernetes sidecar) for open-source models
- **Markdown rendering** (safe, pretty, code blocks styled)
- **Rolling updates** via K8s/Minikube with a live **version pill** that flips after each deploy

> Repo: `github.com/varsilias/zero-downtime`

---

## ✨ Highlights (the clever bits)

- **Zero-downtime “version pill”**  
  A tiny HTMX fragment that refreshes every 2 minutes and updates out-of-band without reloading the page. Perfect visual for rolling updates.
- **Resilient Ollama integration**  
  App detects Ollama at runtime; if unreachable, it **falls back** to an in-memory engine so the demo keeps working.
- **Safe, server-rendered Markdown**  
  `goldmark` + `bluemonday` for beautiful, sanitized Markdown (incl. code blocks) in chat.
- **Production-style build**  
  Multi-stage Docker build compiles Tailwind assets, embeds build info via `-ldflags`, runs on **Distroless non-root**.
- **Operational polish**  
  Request ID middleware, access logs, panic recovery, and a clean `Makefile` that builds, loads into Minikube, sets the image on the Deployment, and waits for rollout.
- **Sidecar pattern for Ollama**  
  Keeps models on a PVC, probes configured, optional auto-pull on start.

---
## 🧱 Architecture

                     ┌────────────────────────────┐
                     │        Frontend            │
                     │  HTMX + Tailwind           │
                     │  - Chat UI                 │
                     │  - Model Selector          │
                     │  - Sessions Sidebar        │
                     └───────────┬────────────────┘
                                 │ (REST + HTMX fragments)
                                 ▼
                     ┌────────────────────────────┐
                     │         Backend (Go)       │
                     │  - API Gateway             │
                     │  - Chat Controller         │
                     │  - Model Manager           │
                     │  - Session Store (mem)     │
                     │  - Logging & Metrics       │
                     └───────────┬────────────────┘
                                 │
             ┌───────────────────┴────────────────────┐
             ▼                                        ▼
     ┌────────────────────────────┐            ┌────────────────────────────┐
     │       Ollama Runtime       │            │       Platform (K8s)       │
     │     (Local or Sidecar)     │            │  - Deployment (app+ollama) │
     │  - /api/version            │            │    RollingUpdate (0/1)     │
     │  - /api/tags               │            │  - Service (80→8080)       │
     │  - /api/generate           │            │  - PVC (ollama-models)     │
     │  - Models on PVC (K8s)     │            │  - Ingress (optional)      │
     └────────────────────────────┘            └────────────────────────────┘

---

## ✅ Features

- Chat UI that feels like a real LLM product:
    - **Sidebar** with recent sessions + **New** chat
    - Chat bubbles with **Markdown** (headings, lists, code fences, inline code)
    - Sticky **top bar** & **composer**, independent scroll areas (sidebar & messages), auto-scroll to last message
    - Immediate **user echo**; assistant bubble after compute
    - Per-response **latency** and timestamp
- **Model dropdown** sourced from Ollama `/api/tags`
- **Admin** endpoint to **pull models** (optional)
- **Version pill** that auto-refreshes every **120s** without htmx loops
- **Health checks** (`/healthz`) and `/version` API with build metadata (version/commit/built_at)
- **Clean logging** via Go `slog` and middleware (Request ID, access log, recoverer)

---

## 🛠 Prerequisites (local dev)

You don’t need all of these at once, but this list avoids “missing tool” surprises:

- **Go** 1.22+
- **Node.js** 20+ and **npm** (for Tailwind build)
- **Air** (optional, hot reload): `go install github.com/air-verse/air@latest`
- **Docker** (to build images)
- **Minikube** + **kubectl** (to demo rolling updates locally)
- **make** (for the Makefile targets)
- **Ollama** (optional, to use a local daemon): https://ollama.com
- **jq** (optional, nicer curls)

> Apple Silicon ✅. For CPU inference, use small models and keep `num_ctx` modest.

---

## 🚀 Run locally (no Docker)

1) **Install Go & UI deps**
```bash
go mod download
npm ci
```
2) **Build Tailwind CSS**
```bash
# Development:
npm run tw:build
# Production (minified):
npm run tw:prod
```
3) **Start the server**
```bash
# Plain:
go run .

# With hot reload (Air) & build info flags via script:
air -c .air.toml
```
4) **Open**
- App: `http://localhost:8080`
- Build info: `http://localhost:8080/version`
- Health: `http://localhost:8080/healthz`

**Optional: Use your local Ollama**
```bash
# In another terminal:
ollama serve &
ollama pull gemma3:270m

# Run app without startup wait (dev-friendly):
OLLAMA_WAIT=false OLLAMA_BASE_URL=http://localhost:11434 go run .
```
Pick a model from the dropdown and chat.

---

## 🐳 Docker build & run
```bash
# Build with ldflags baked in (Makefile also sets these)
docker build -t varsilias/zero-downtime:dev .
docker run -p 8080:8080 varsilias/zero-downtime:dev
```
**Dockerfile highlights**
- Node stage builds Tailwind → web/static/dist/app.css
- Go stage builds with -ldflags & -trimpath (CGO_ENABLED=0)
- Distroless non-root runtime; templates/assets copied into /app/web

---

## ☸️ Kubernetes (Minikube) — zero-downtime demo
> First rollout downloads models (PVC-backed cache). Subsequent rollouts are fast.
> 
1. Start Minikube with headroom
```bash
minikube start --memory=12000 --cpus=4
```
2. Apply PVC for Ollama models
```bash
kubectl apply -f k8s/pvc.yaml
```
3. Build + load + rollout
```bash
make release
# does: docker build → minikube image load → apply svc/deploy → set image → rollout status → print URL 
```
4. Open the printed URL. Watch the version pill.
5. Do another release (and watch zero-downtime flip)
```bash
VERSION=demo-$(date +%s) make release 
```

**Ollama sidecar notes**
- Sidecar image: ollama/ollama listening on :11434
- Models persisted in PVC ollama-models
- App talks to http://127.0.0.1:11434 inside the Pod
- Optional env on sidecar: OLLAMA_PULL_MODELS="gemma3:270m smollm:135m" to auto-pull at start
- On CPU, serialize requests: OLLAMA_NUM_PARALLEL=1
---
## 🔧 Environment variables

| Var                    | Default                  | Purpose                                                        |
| ---------------------- | ------------------------ | -------------------------------------------------------------- |
| `ADDR`                 | `:8080`                  | HTTP bind address                                              |
| `LOG_LEVEL`            | `info`                   | `debug` \| `info` \| `warn` \| `error`                         |
| `LOG_JSON`             | `true`                   | JSON logs (set `false` for pretty text)                        |
| `OLLAMA_BASE_URL`      | `http://localhost:11434` | Ollama API base (or `http://127.0.0.1:11434` for sidecar)      |
| `OLLAMA_WAIT`          | `true`                   | On startup, wait for Ollama/models. Set `false` for local dev. |
| `OLLAMA_WAIT_TIMEOUT`  | `180s`                   | Max time to wait before continuing anyway                      |
| `OLLAMA_WAIT_INTERVAL` | `2s`                     | Poll frequency during startup wait                             |
| `OLLAMA_WAIT_MODELS`   | `"gemma3:270m smollm:135m deepseek-r1:1.5b"`                | Space-separated list: `gemma3:270m smollm:135m`                |

---

## 🔌 API (quick reference)
- `POST /api/chat` → chat with selected model
```bash
{ "model":"gemma3:270m", "message":"Hello!" } 
```
- `GET /api/models → ["gemma3:270m","smollm:135m","deepseek-r1:1.5b", ...]`
- `GET /api/history/:session_id` → chat transcript (in-memory)
- `POST /admin/models/pull → { "name": "gemma3:270m" }` (optional admin)
- `GET /version → { "version": "...", "commit": "...", "built_at": "..." }`

**UI endpoints**
- `GET /` – chat UI

- `POST /ui/chat` – HTMX post (returns user + assistant bubbles)

- `POST /ui/session/new` – creates a new session (via HX-Redirect)

- `GET /ui/version-pill` – HTMX fragment for the version pill (polled by a non-swapped element every 120s)
---
## Why is this project awesome?
> Production-flavored design, real operational concerns addressed, clean Go layering, elegant UI without SPA, and a compelling rolling-update strategy.
- Zero-Downtime in 60 seconds: run make release twice and point at the version pill. No 5xx, pods roll gracefully, traffic keeps flowing.

- Sidecar runtime pattern: clean in-pod service-to-service without extra networking; PVC-backed model cache; probes; resource requests/limits; startup wait with timeout.

- Safe, server-rendered UI: modern-feeling app without a SPA—HTMX requests, partial swaps, Markdown rendered on the server, sanitized with bluemonday.

- Thoughtful DX: Air hot reload wired with -ldflags; Tailwind build baked into Docker; Distroless runtime; request IDs; access logs; panic recovery.

- Pragmatic fallbacks: if Ollama isn’t reachable, the app still works, preserving the demo and UX.

---

# Limitations (By Design)

- **In-memory sessions** only (no DB yet) — sidebar lists current-run chats; restart loses history.
- **No token streaming** yet (responses are returned whole; no SSE/WebSocket).
- **No auth/tenancy** — endpoints are open; fine for demos, not for production.
- **Basic backpressure** — no rate limiting; rely on ingress/gateway if needed.
- **CPU inference defaults** — recommended to use small models; larger models need GPU/tuning.
- **Model pulls can be long** — use longer timeouts or a background pull job for big models.

> These choices keep the demo lean and make the zero-downtime story the star.

---

# Roadmap (If This Weren’t a Demo)

- **Persistence**: Postgres for sessions/users/model metadata; S3 for attachments.
- **Streaming responses**: SSE/WebSocket with token-by-token updates + cancelation.
- **Observability**: Prometheus metrics (latency, tokens/sec), OpenTelemetry traces, Grafana dashboards.
- **Auth & RBAC**: login, API keys, roles; protect admin endpoints.
- **Background workers**: queued model pulls, warm-ups, summarization jobs.
- **GPU & autoscaling**: HPA on latency/QPS; dedicated GPU node pool; model-aware scheduling.
- **Config & GitOps**: ConfigMaps for model lists, ArgoCD, image updater; canary rollouts.
- **Testing**: e2e smoke (health, version, chat flow), load testing (vegeta/k6), chaos (pod kill) to prove resilience.

---

# Troubleshooting

### CSS 404 / Wrong MIME
- Ensure `web/static/dist/app.css` exists (`npm run tw:prod`).
- Static server must point at `web/static`; link `/static/app.css`.

### HTMX Polling Loops
- Don’t use `hx-trigger="load, every …"` with `hx-swap="outerHTML"` on the **same** element.
- Use a hidden **poller** targeting `#version-pill`, or OOB swap.

### Nil Map Panic (MemoryStore)
- Initialize via `NewMemoryStore()` or guard with `ensure()` before writes.
- Pass the store **by pointer** across the app.

### Model Pulls Never Finish
- Use a **no-timeout** HTTP client or long context for `/api/pull`.
- Consider pulling via sidecar `postStart` or a background job.

### 500 on `/api/generate`
- If first call after model load, retry with small backoff.
- Keep `options` light on CPU: `num_ctx:512`, `num_predict:128`, `num_thread:4`, `num_gpu_layers:0`.
- Increase server `WriteTimeout` (≥ 3–5 min); serialize requests: `OLLAMA_NUM_PARALLEL=1`.
- Check sidecar resources (CPU/mem) and Minikube size.

### Version Pill Doesn’t Update
- Ensure polling element is **not swapped** itself.
- Endpoint `/ui/version-pill` must send `Cache-Control: no-store`.
- Poll interval set to `every 120s`.

### Kubernetes Rollout Stalls
- Confirm `readinessProbe` paths and ports.
- `maxUnavailable: 0` requires enough capacity for `maxSurge: 1`.
- Check PVC events if sidecar waits on model cache.

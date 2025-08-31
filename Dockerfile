# ── ASSETS ─────────────────────────────────────────────────────────────────────
FROM node:20-alpine AS assets
WORKDIR /app
COPY package.json package-lock.json* ./
RUN npm ci --no-audit --no-fund
COPY web/static ./web/static
COPY web/templates ./web/templates

# Build tailwind -> web/static/dist/app.css
RUN npm run tw:prod

# ── GO BUILD ───────────────────────────────────────────────────────────────────
FROM golang:1.22-alpine AS builder
WORKDIR /app

# Install git for 'go mod' and allow private modules
RUN apk update && \
    apk add --no-cache git ca-certificates

COPY go.mod go.sum ./
RUN go mod download

# Copy the whole source (except what's ignored by .dockerignore)
COPY . .
# Bring in built assets from the Node stage
COPY --from=assets /app/web/static/dist ./web/static/dist

# Build args for linker flags (set these at build time with --build-args)
ARG VERSION=dev
ARG COMMIT=none
ARG BUILT_AT=unknown

# IMPORTANT: module path (from go.mod) + package path for buildinfo
# module github.com/varsilias/zero-downtime
# package is internal/buildinfo
ENV PKG_PATH=github.com/varsilias/zero-downtime/internal/buildinfo

# Static build (distroless friendly), smaller binary
ENV CGO_ENABLED=0
RUN go build -trimpath -ldflags "\
  -s -w \
  -X ${PKG_PATH}.Version=${VERSION} \
  -X ${PKG_PATH}.Commit=${COMMIT} \
  -X ${PKG_PATH}.BuiltAt=${BUILT_AT}" \
  -o /out/app .

# ── RUNTIME ────────────────────────────────────────────────────────────────────
FROM gcr.io/distroless/static:nonroot
WORKDIR /app
COPY --from=builder /out/app /app/app
# templates + assets must be present at runtime (UI reads from disk)
COPY --from=builder /app/web /app/web
EXPOSE 8080
USER nonroot:nonroot
ENV ADDR=8080 LOG_LEVEL=info LOG_JSON=false
ENTRYPOINT ["/app/app"]

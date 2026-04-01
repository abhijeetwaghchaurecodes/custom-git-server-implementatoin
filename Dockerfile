# syntax=docker/dockerfile:1
# ═══════════════════════════════════════════════════════════════════════════════
# SEBI Mutual Fund Backend — Multi-Stage Docker Build
# ═══════════════════════════════════════════════════════════════════════════════
#
# This Dockerfile implements a 3-stage build pipeline:
#
#   Stage 1 (inject)  — Runs the service-injector to generate bootstrap.go
#                        with all 4 MF implementations wired in from project2.
#
#   Stage 2 (build)   — Compiles project1-mf-backend (now with real bootstrap.go)
#                        into a static Linux binary. All implementations are
#                        resolved at compile time — no runtime injection needed.
#
#   Stage 3 (runtime) — Minimal Debian image with just the binary + CA certs.
#                        Pull this on ANY machine, run it, hit the APIs.
#
# Build context: the mf-sebi/ root directory (contains go.work + all 3 modules)
#
# Usage:
#   docker build -t sebi-mf-backend:latest .
#   docker run -p 8080:8080 sebi-mf-backend:latest
#
# ═══════════════════════════════════════════════════════════════════════════════

# ── Stage 1: INJECT ───────────────────────────────────────────────────────────
# Runs the service-injector to:
#   1. Seed project2 implementations into SQLite
#   2. Validate via Yaegi smoke-test
#   3. Render bootstrap.go via Gonja template
#   4. Output: project1-mf-backend/internal/bootstrap/bootstrap.go (populated)

FROM golang:1.21-bookworm AS inject

WORKDIR /workspace

# Copy the entire workspace (go.work + all 3 modules + templates)
COPY go.work ./
COPY project1-mf-backend/ ./project1-mf-backend/
COPY project2-mf-implementations/ ./project2-mf-implementations/
COPY service-injector/ ./service-injector/
COPY templates/ ./templates/

# Download dependencies for ALL modules in the workspace
RUN cd service-injector && go mod download
RUN cd project1-mf-backend && go mod download
RUN cd project2-mf-implementations && go mod download

# Run the injector — this generates bootstrap.go with all 4 implementations
# wired into project1's registry
WORKDIR /workspace/service-injector
RUN go run ./cmd/injector

# Verify the generated bootstrap.go exists and has content
RUN echo "=== Generated bootstrap.go ===" && \
    cat /workspace/project1-mf-backend/internal/bootstrap/bootstrap.go && \
    echo "=== Injection complete ==="


# ── Stage 2: BUILD ────────────────────────────────────────────────────────────
# Compiles the fully-wired project1-mf-backend into a static binary.
# Because bootstrap.go now imports project2 packages directly, the Go compiler
# pulls in all 4 MF implementations at compile time.

FROM golang:1.21-bookworm AS build

WORKDIR /workspace

# Copy the entire workspace again (with the generated bootstrap.go from Stage 1)
COPY --from=inject /workspace/ ./

# Build the backend binary
# CGO_ENABLED=0 for static linking — no libc dependency in the final image
# -ldflags="-s -w" strips debug info for a smaller binary
WORKDIR /workspace/project1-mf-backend
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /out/sebi-mf-backend ./cmd/server

# Verify the binary was produced
RUN ls -lh /out/sebi-mf-backend && \
    echo "Binary size: $(du -h /out/sebi-mf-backend | cut -f1)"


# ── Stage 3: RUNTIME ──────────────────────────────────────────────────────────
# Minimal production image. No Go toolchain, no source code, no SQLite.
# Just the compiled binary + CA certificates for outbound HTTPS.

FROM debian:bookworm-slim AS runtime

WORKDIR /app

# Install CA certificates (needed if the app makes outbound HTTPS calls)
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates curl && \
    rm -rf /var/lib/apt/lists/*

# Create non-root user for security
RUN groupadd -r sebiapp && useradd -r -g sebiapp -s /usr/sbin/nologin sebiapp

# Copy the compiled binary from the build stage
COPY --from=build /out/sebi-mf-backend /app/sebi-mf-backend

# Set ownership
RUN chown -R sebiapp:sebiapp /app

USER sebiapp

# The Echo server listens on :8080
EXPOSE 8080

# Health check — hits the /health endpoint
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/sebi-mf-backend"]

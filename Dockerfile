# ── Stage 1: build ────────────────────────────────────────────────────────────
FROM golang:1.25-alpine AS builder

# Install git (needed for go mod download with VCS stamps)
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Cache dependency downloads separately from source copy
COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Build a fully static binary (CGO disabled, strip debug symbols)
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build \
    -ldflags="-s -w -extldflags '-static'" \
    -o /build/api \
    ./cmd/api/main.go

# ── Stage 2: hardened runtime ─────────────────────────────────────────────────
# gcr.io/distroless/static-debian12: no shell, no package manager, minimal attack surface
FROM gcr.io/distroless/static-debian12:nonroot

# Copy CA certs and timezone data from builder so TLS calls and time zones work
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Copy the binary
COPY --from=builder /build/api /api

# The distroless:nonroot image already runs as uid 65532 (nonroot).
# Declare this explicitly so Kubernetes securityContext checks pass.
USER nonroot:nonroot

EXPOSE 8080

ENTRYPOINT ["/api"]

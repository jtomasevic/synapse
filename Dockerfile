# =============================================================================
# Synapse API — Multi-stage Dockerfile
# =============================================================================
# Stage 1: Build the Go binary
# Stage 2: Minimal runtime image
# =============================================================================

# -- Build stage --------------------------------------------------------------
FROM golang:1.24-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Cache dependency downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /synapse-api ./cmd/synapse-api

# -- Runtime stage ------------------------------------------------------------
FROM alpine:3.20

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /synapse-api /app/synapse-api
COPY config.docker.yaml /app/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/synapse-api", "-config", "/app/config.yaml"]

FROM golang:1.22-alpine AS builder

WORKDIR /app

# Copy dependency manifests first (layer cache)
COPY go.mod go.sum ./
RUN go mod download

# Copy source
COPY . .

# Build static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /bin/inventory-manage ./cmd/server

# ── Runtime image ──────────────────────────────────────────────────────────────
FROM alpine:3.19

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/inventory-manage .
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080

ENTRYPOINT ["./inventory-manage"]

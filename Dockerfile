FROM golang:1.25.9-alpine3.22 AS builder

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
FROM alpine:3.22

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /bin/inventory-manage .
COPY --from=builder /app/migrations ./migrations
COPY --from=builder /app/config ./config

EXPOSE 8080

ENTRYPOINT ["./inventory-manage"]

# Traefik API Gateway — Setup & Reference

> **Version:** 1.0 | **Task:** INV-SPR03-TASK-008  
> Traefik v2.11 acts as the single entry point for all HTTP traffic to the Inventory Management backend.

---

## Architecture

```
Client
  │
  ▼
┌──────────────────────────────┐
│   Traefik (:80)              │  ← Single public entrypoint
│   API Gateway / Reverse Proxy│
└──────────────┬───────────────┘
               │  Routes by rule
               ▼
┌──────────────────────────────┐
│   inventory_app (:8080)      │  ← Go backend (not exposed externally)
│   Golang + Gin               │
└──────────────────────────────┘
```

---

## Quick Start

### 1. Development (infrastructure only, backend runs locally)

```bash
# Start DB + MQTT + Redis + Traefik (but NOT the app container)
docker compose up -d

# Run the backend locally — Traefik cannot route to localhost from inside Docker
# when running this way, access backend directly at: http://localhost:8080
make run
```

> **Note:** When the Go app runs locally (not in Docker), Traefik cannot route to it because Traefik uses Docker service discovery. Use the full Docker stack (see below) to test Traefik routing.

### 2. Full Docker Stack (Traefik routing enabled)

```bash
# Build and start all services including app + Traefik
docker compose --profile app up --build -d

# Verify all containers are running
docker compose ps

# Test routing through Traefik
curl http://localhost/health
curl http://localhost/api/v1/devices
```

### 3. Stop all services

```bash
docker compose --profile app down
```

---

## Routing Table

| Path Pattern | Middleware | Backend | Port |
|-------------|-----------|---------|------|
| `/api/*` | RateLimit (100 req/s) | inventory_app | 8080 |
| `/health` | — | inventory_app | 8080 |

---

## Traefik Dashboard

**URL:** `http://localhost:8081/dashboard/`

> ⚠️ **Local dev only** — the dashboard is running in **insecure mode** (no authentication).  
> Never expose port 8081 to a public network or production environment.

**What you can see:**
- All registered routers and their rules
- All services and their health status
- All active middlewares
- Real-time traffic overview

---

## Rate Limiting (AC-06)

Configured via Docker labels on the `app` service:

| Parameter | Value | Meaning |
|-----------|-------|---------|
| `average` | 100 | Max 100 requests per period |
| `burst` | 50 | Allow burst up to 50 extra requests |
| `period` | 1s | Reset period = 1 second |

When the rate limit is exceeded, Traefik returns **HTTP 429 Too Many Requests**.

---

## Configuration Files

| File | Purpose |
|------|---------|
| `traefik/traefik.yml` | Static config: entrypoints, providers, dashboard |
| `docker-compose.yml` | Dynamic config: routing rules via service labels |

### How dynamic config works (labels)

Traefik reads Docker labels at runtime — no restart needed for label changes:

```yaml
labels:
  - "traefik.enable=true"
  - "traefik.http.routers.inventory-api.rule=PathPrefix(`/api`)"
  - "traefik.http.routers.inventory-api.middlewares=api-ratelimit"
  - "traefik.http.middlewares.api-ratelimit.ratelimit.average=100"
```

---

## Troubleshooting

### Traefik dashboard shows no routes
- Check that the `app` container is running: `docker compose ps`
- Verify `traefik.enable=true` label is present on `inventory_app`
- Verify `traefik.docker.network=inventory_net` matches the compose network name

### curl http://localhost/health returns 404
- The router rule must match exactly — check `Path(/health)` vs actual health endpoint path
- View Traefik logs: `docker logs inventory_traefik`

### Port 80 already in use
```bash
# Find what's using port 80
sudo lsof -i :80
# Stop it or change Traefik entrypoint to another port in traefik/traefik.yml
```

### Port 8081 already in use
Change the dashboard port in `docker-compose.yml`:
```yaml
ports:
  - "8082:8081"   # host:container
```

---

## Production Checklist (NOT for local dev)

> For production deployment, the following changes MUST be made:

```
[ ] Remove insecure dashboard (set api.insecure: false, add auth middleware)
[ ] Add TLS/HTTPS via Let's Encrypt (acme provider in traefik.yml)
[ ] Restrict dashboard access to internal network only
[ ] Set log.level: WARN (not INFO/DEBUG)
[ ] Mount persistent certificate storage volume
[ ] Use environment secrets for any credentials (never hardcode)
```

---

*Managed by: INV-SPR03-TASK-008 | Traefik v2.11 | Inventory Management System*

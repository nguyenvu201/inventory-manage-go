# pkg — Shared Utilities

This directory contains **reusable, project-agnostic** utility packages.

## Rules

- Packages here must have **zero dependency** on `internal/` code
- They must be usable independently in other projects
- Do NOT put business logic here — that belongs in `internal/usecase/`

## Current packages

_(Empty — packages will be added as shared utilities are identified during development)_

## Examples of what belongs here

| Package | Purpose |
|---------|---------|
| `pkg/respond/` | HTTP JSON response helpers |
| `pkg/telemetrylog/` | Zerolog helper for structured IoT logging |
| `pkg/pagination/` | Generic pagination structs |

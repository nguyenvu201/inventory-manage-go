---
description: Developer workflow for implementing Golang tasks in the Inventory Management System
---

# /golang-developer — Developer Task Implementation Workflow

Use this workflow when you are assigned a task (`🔄 IN_PROGRESS`) and need to implement it following FDA 21 CFR Part 11 / IEC 62304 standards.

---

## Step 1 — Read Your Task

```bash
# Open the task registry and find your IN_PROGRESS task
cat docs/sprints/task_registry.md
```

1. Note the **Task ID** (e.g., `INV-SPR01-TASK-001`)
2. Open the corresponding sprint file
3. Read the **entire task block**: Description, all ACs, Related Technologies, Dependencies
4. Do NOT skip any AC — each one is a verifiable requirement

> If the task status is `✅ APPROVED` but not yet `🔄 IN_PROGRESS`, update the status first before starting.

---

## Step 2 — Update Task: APPROVED → IN_PROGRESS

In the sprint file, update the task header and add a Status History row:

```markdown
> **Status:** 🔄 IN_PROGRESS
> **Assignee:** Developer

| Date       | From     | To          | Performed by | Notes                    |
|------------|----------|-------------|--------------|--------------------------|
| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer    | Started implementation   |
```

Also update `docs/sprints/task_registry.md` — change the status column for this task.

---

## Step 3 — Set Up the Development Environment

```bash
# Start all infrastructure services (Postgres, Redis, MQTT)
make docker-up

# Verify all services are healthy
docker-compose ps

# Run pending migrations
make migrate

# Run the service
make run
```

If `make` targets don't exist yet (Sprint 1), create `Makefile` as part of AC-05.

---

## Step 4 — Implement Following the AC Checklist

Work through ACs **in order**. For each AC:

1. Write the test first (TDD preferred)
2. Implement the code to make the test pass
3. Tick the AC when verified: `- [x] AC-01: ...`

### Project Structure Reminder

```
cmd/server/main.go        ← entry point (minimal, calls initialize.Run())
config/local.yaml         ← configuration (Viper)
internal/
  model/                  ← models (no external deps)
  service/impl/           ← business logic implementations
  repository/postgres/    ← DB implementations
  controller/             ← Gin HTTP handlers
  routers/                ← Gin router groups
  worker/                 ← background workers (MQTT, cron)
migrations/               ← golang-migrate SQL files
```

### Before writing any code, ask:
- Which layer does this belong to? (model / service / repository / controller)
- Does the interface belong in `service/interface.go`? (yes — always)
- Am I about to hardcode a string? (no — use local.yaml / Config)
- Am I ignoring an error? (no — always wrap with `fmt.Errorf`)
- Am I checking for `global.Rdb != nil` before using Redis? (yes - always check)

---

## Step 5 — Write Tests

### For every new function/service:
```bash
# Run tests continuously while developing
go test ./internal/... -v -run TestMyFunction

# Check coverage
go test ./internal/... -cover
```

### Required before PR:
```bash
make test         # All tests pass
make test-race    # No race conditions
make lint         # No vet/staticcheck warnings
```

Coverage ≥ 80% for all business logic (use cases, domain services, validators).
Make sure your gin controller tests use `httptest.NewRecorder()` and correctly mock services.

---

## Step 6 — Submit PR: IN_PROGRESS → IN_REVIEW

### Before creating the PR, run the full checklist:

```
[ ] All ACs are ticked [x] in the sprint file
[ ] go test ./... passes
[ ] go test -race -count=1 ./... passes
[ ] go vet ./... clean
[ ] staticcheck ./... clean
[ ] No hardcoded secrets
[ ] Every error wrapped with fmt.Errorf("context: %w", err)
[ ] Every log entry has device_id + trace_id, using global.Logger
[ ] All DB tables have migration files
[ ] Interfaces defined in service/, not repository/
[ ] Unit test coverage ≥ 80% for business logic
[ ] Integration tests written for repository layer
```

### Update task status in sprint file:

```markdown
> **Status:** 👀 IN_REVIEW

| Date       | From        | To        | Performed by | Notes                          |
|------------|-------------|-----------|--------------|-------------------------------|
| YYYY-MM-DD | IN_PROGRESS | IN_REVIEW | Developer    | PR #XX — all ACs implemented  |
```

Update `docs/sprints/task_registry.md` as well.

### Ping QA Agent:
At the very end of your response, you MUST ping the QA agent by writing:
`@[/golang-tester] I have finished implementing the task and moved it to IN_REVIEW. Please check.`

### Commit convention:
```bash
git add .
git commit -m "feat(INV-SPR01-TASK-001): setup golang project structure and docker compose"
```

Format: `<type>(<task-id>): <short description>`

Types: `feat` | `fix` | `test` | `refactor` | `docs` | `chore`

---

## Step 7 — After Review

### If REJECTED ❌
- Read the reviewer's feedback carefully
- Fix all failing ACs
- Do NOT change the task ID or delete history rows
- Change status back to `🔄 IN_PROGRESS`, add history row
- Re-submit → `👀 IN_REVIEW`

### If VERIFIED ✅ → CLOSED 🔒
- Lead will update the task status to `🔒 CLOSED`
- Move to the next `✅ APPROVED` task in the sprint
- Check sprint dependency order before starting next sprint

---

## Important File Paths

| File | When to use |
|------|-------------|
| `docs/sprints/task_registry.md` | Find your task, update status |
| `docs/sprints/sprint_0N_*.md` | Read full AC list, update status history |
| `.agents/rules/golang-developer-rules.md` | Coding standards reference |
| `docs/workflows/ba_task_creation_workflow.md` | Understand full FDA lifecycle |
| `docs/sprints/_overview.md` | Sprint progress at a glance |
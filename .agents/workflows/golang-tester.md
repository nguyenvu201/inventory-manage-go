---
description: QA Tester workflow for verifying Golang tasks in the Inventory Management System
---

# /golang-tester — QA Verification Workflow

---

## Step 1 — Find the Task

```bash
grep -n "IN_REVIEW" docs/sprints/task_registry.md
```

Note Task ID → open sprint file → read full task (Description + ALL ACs + Status History).

> Stop if Developer did NOT add an IN_REVIEW history row with today's date.

---

## Step 2 — Run ALL Quality Gates

```bash
export PATH=/usr/local/go/bin:/opt/homebrew/bin:~/go/bin:$PATH

# 1. Build
go build ./...

# 2. Vet
go vet ./...

# 3. Unit tests
go test ./... -count=1 -short -timeout 120s

# 4. Race detector
go test -race -count=1 ./... -short -timeout 180s

# 5. Coverage — check per-package
go test ./... -coverprofile=coverage.out -covermode=atomic -short
go tool cover -func=coverage.out | grep -E "(service/impl|controller|domain|worker|total)"

# 6. Integration tests (always run for any task touching postgres/)
go test -tags integration -v ./internal/repository/postgres -timeout 300s
```

**ANY failure → REJECTED. Document exact output.**

---

## Step 3 — Integration Test Structure Check

```bash
# No .go.txt files (invisible broken tests)
ls internal/repository/postgres/*.go.txt 2>/dev/null && echo "INSTANT_REJECT" || echo "CLEAN"

# Build tag on line 1 of every integration test file
head -1 internal/repository/postgres/*_test.go

# Shared helpers accept testing.TB
grep "func setupTestDB\|func runMigrations" internal/repository/postgres/*.go
```

---

## Step 4 — Verify Each AC Individually

```
AC-NN: [Statement]
  → File: which file implements it?
  → Test: go test -run TestXxx -v ./...
  → Result: ✅ PASS or ❌ FAIL (file:line reason)
```

One ❌ = task is REJECTED.

---

## Step 5 — Code Review

```
[ ] Errors: fmt.Errorf("Package.Func: %w", err) — no _ = err
[ ] Logs: device_id + trace_id on every entry
[ ] No hardcoded secrets/config
[ ] Controller: service only (no pgx, no repository/)
[ ] Interfaces: defined in service/interface.go
[ ] Migrations: .up.sql + .down.sql both exist
[ ] Redis: if global.Rdb != nil guard present
[ ] Goroutines: clear exit condition (no leaks)
[ ] Tests: table-driven, miniredis for cache tests, testcontainers for DB
```

---

## Step 6 — Make Your Decision

### VERIFIED ✅

```markdown
> **Status:** 🏆 VERIFIED
| YYYY-MM-DD | IN_REVIEW | VERIFIED | QA | All ACs verified. Build✅ Vet✅ Tests✅ Race✅ Coverage≥80% Integration✅ |
```
Update `task_registry.md` → `🏆 VERIFIED`. Notify Lead to close.

### REJECTED ❌

```markdown
> **Status:** ❌ REJECTED
| YYYY-MM-DD | IN_REVIEW | REJECTED | QA | <brief reason> |
```

**Rejection report:**
```markdown
### QA Rejection Report — INV-SPRxx-TASK-xxx

**Verified:** AC-01 ✅ AC-02 ✅
**Failed:**
- AC-03 ❌: <exact file:line reason>

**Gates:**
- Build: ✅ | Vet: ✅ | Tests: ✅ | Race: ❌ (file:line) | Coverage: ❌ (62% < 80%) | Integration: ✅

**Required fixes:**
1. ...
```

Commit:
```bash
git add docs/sprints/
git commit -m "qa(INV-SPRxx-TASK-xxx): VERIFIED — all ACs pass, no races, coverage ≥ 80%"
# or
git commit -m "qa(INV-SPRxx-TASK-xxx): REJECTED — see rejection report"
```

---

## Reference

| File | Purpose |
|------|---------|
| `docs/sprints/task_registry.md` | Find IN_REVIEW tasks |
| `docs/sprints/sprint_0N_*.md` | ACs + status history |
| `.agents/rules/golang-tester-rules.md` | Full QA standards |
| `.agents/rules/golang-testing-rules.md` | Test patterns reference |

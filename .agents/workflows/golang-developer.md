---
description: Developer workflow for implementing Golang tasks in the Inventory Management System
---

# /golang-developer — Developer Task Implementation Workflow

---

## Step 1 — Read Your Task

```bash
grep -n "IN_PROGRESS\|APPROVED" docs/sprints/task_registry.md
cat docs/sprints/sprint_0N_*.md | grep -A 50 "TASK-XXX"
```

Read **ALL ACs** before writing any code. Note dependencies.

---

## Step 2 — Update Status: APPROVED → IN_PROGRESS

```markdown
> **Status:** 🔄 IN_PROGRESS

| YYYY-MM-DD | APPROVED | IN_PROGRESS | Developer | Started implementation |
```

Update `docs/sprints/task_registry.md` too.

---

## Step 3 — Set Up Environment

```bash
make docker-up   # TimescaleDB + Redis + MQTT
make migrate     # run pending migrations
make run         # start server (verify it boots without error)
```

---

## Step 4 — Implement (TDD Order)

For each AC:
1. Write the failing test
2. Implement the code to pass the test
3. Tick AC: `- [x] AC-01: ...`

**Before writing any code, ask:**
- Which layer? (model / service / repository / controller)
- Does interface belong in `service/interface.go`? → Yes, always
- Am I hardcoding anything? → No, use `local.yaml` / `global.Config`
- Am I ignoring an error? → No, always `fmt.Errorf("Ctx: %w", err)`
- Am I using Redis without nil check? → Always `if global.Rdb != nil`

---

## Step 5 — Run Quality Gates

```bash
export PATH=/usr/local/go/bin:/opt/homebrew/bin:~/go/bin:$PATH

# Must ALL pass before submitting:
go build ./...
go vet ./...
go test ./... -short -count=1
go test -tags integration ./internal/repository/postgres -timeout 300s
go test -race -count=1 ./... -short
go test ./... -coverprofile=coverage.out -short && go tool cover -func=coverage.out
```

**Integration test integrity checks:**
```bash
# No .go.txt files (broken, invisible to compiler)
ls internal/repository/postgres/*.go.txt 2>/dev/null && echo "FAIL" || echo "CLEAN"

# All integration files have build tag line 1
head -1 internal/repository/postgres/*_test.go

# Shared helpers use testing.TB
grep "func setupTestDB\|func runMigrations" internal/repository/postgres/*.go
```

---

## Step 6 — Submit: IN_PROGRESS → IN_REVIEW

### Pre-PR Checklist (self-sign before pinging QA)

```
── Code ─────────────────────────────────────────────────────────────
[ ] All ACs ticked [x] in sprint file
[ ] No hardcoded secrets / connection strings
[ ] All errors wrapped: fmt.Errorf("Package.Func: %w", err)
[ ] All logs: device_id + trace_id, global.Logger (zap)
[ ] Interfaces in service/interface.go (not repository/)
[ ] Redis: if global.Rdb != nil { ... }
[ ] New tables: .up.sql + .down.sql migrations

── Tests ────────────────────────────────────────────────────────────
[ ] Unit tests: table-driven, file header with Task ID + AC map
[ ] Controller tests: httptest.NewRecorder() + gin.SetMode(TestMode)
[ ] Integration test: testcontainers real DB
[ ] global.Logger initialized in tests (zap.NewNop())
[ ] Redis caching: MISS + HIT + STORE paths covered (miniredis)
[ ] FK deps seeded before table under test
[ ] Integration files: //go:build integration on LINE 1
[ ] Benchmark files: //go:build integration on LINE 1 (if uses setupTestDB)
[ ] Shared helpers: testing.TB not *testing.T
[ ] UUID table tests: no hardcoded string IDs — use DB RETURNING
[ ] No .go.txt files

── Gates ────────────────────────────────────────────────────────────
[ ] go build ./...                                        → 0 errors
[ ] go vet ./...                                          → 0 warnings
[ ] go test ./... -short                                  → all pass
[ ] go test -tags integration ./internal/repository/...  → all pass
[ ] go test -race ./... -short                            → 0 races
[ ] Coverage ≥ 80% per-package (model/, domain/ ≥ 90%)
[ ] Swagger annotations on all new controller methods
```

### Update sprint file + task_registry.md:

```markdown
> **Status:** 👀 IN_REVIEW
| YYYY-MM-DD | IN_PROGRESS | IN_REVIEW | Developer | All gates pass. Coverage ≥ 80%. |
```

### Commit:
```bash
git add .
git commit -m "feat(INV-SPR03-TASK-004): implement historical reporting API with Redis cache"
```

### Ping QA:
```
@[/golang-tester] please review INV-SPR03-TASK-004
```

---

## Step 7 — After Review

### If REJECTED ❌
1. Read rejection report carefully — fix ALL items listed
2. Do NOT change Task ID or delete history rows
3. Update status → `🔄 IN_PROGRESS`, add history row
4. Fix → re-run all gates → update status → `👀 IN_REVIEW`
5. Ping QA again

### If VERIFIED ✅
- Lead closes → `🔒 CLOSED`
- Move to next `✅ APPROVED` task

---

## Reference

| File | Purpose |
|------|---------|
| `docs/sprints/task_registry.md` | Task list + status |
| `docs/sprints/sprint_0N_*.md` | AC list + history |
| `.agents/rules/golang-developer-rules.md` | Coding standards |
| `.agents/rules/golang-testing-rules.md` | Test patterns |
| `docs/sprints/_overview.md` | Sprint progress |
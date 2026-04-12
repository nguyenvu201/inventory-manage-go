---
description: QA Tester workflow for verifying Golang tasks in the Inventory Management System
---

# /golang-tester — QA Verification Workflow

Use this workflow when a task is `👀 IN_REVIEW` and you need to verify it to **VERIFIED** or **REJECTED** following FDA 21 CFR Part 11 / IEC 62304 standards.

---

## Step 1 — Find the Task Under Review

```bash
# Find all tasks currently in IN_REVIEW
grep -n "IN_REVIEW" docs/sprints/task_registry.md
```

1. Note the **Task ID** (e.g., `INV-SPR01-TASK-001`)
2. Open the sprint file → read the **full task block**: Description, ALL ACs, Status History
3. Check the Status History: Developer must have an `IN_REVIEW` row with today's date

> If Status History is missing the `IN_REVIEW` row → stop, ask Developer to update audit trail first.

---

## Step 2 — Check Out the Code

Identify all files changed for this task. Use git to see what was implemented:

```bash
# View files changed since last review commit
git log --oneline -5
git show --stat HEAD        # See files in last commit

# Or diff against a specific commit
git diff <base-commit> --name-only
```

For each changed file, note which AC it implements.

---

## Step 3 — Run All Quality Gates

**You MUST run ALL of these. Do not skip any.**

```bash
# 1. Build — zero errors
go build ./...

# 2. Vet — zero warnings
go vet ./...

# 3. All tests pass
go test ./... -v -count=1 -timeout 120s

# 4. Race detector — zero races
go test -race -count=1 ./... -timeout 180s

# 5. Coverage — must be ≥ 80% for business logic packages
go test ./... -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out

# 6. Staticcheck (if installed)
staticcheck ./...
```

**If ANY gate fails → REJECTED immediately. Document the exact failure.**

---

## Step 4 — Verify Each AC Individually

Go through every AC in the sprint file **one by one**:

```
AC-01: <Read the statement>
  → Find: which file/function implements this?
  → Test: which test covers it?
  → Run: go test ./... -run TestXxx -v
  → Result: ✅ PASS or ❌ FAIL + reason
```

For the current active task `INV-SPR01-TASK-001`, check each AC:

| AC | What to verify | Where |
|----|---------------|-------|
| AC-01 | `go.mod` exists, correct module name, all dirs present: `cmd/`, `internal/`, `pkg/`, `config/`, `migrations/` | `go.mod`, directory structure |
| AC-02 | `docker-compose.yml` has 3 services: `db` (timescaledb), `mosquitto`, `app`; healthchecks present | `docker-compose.yml` |
| AC-03 | Migration file creates `raw_telemetry` as hypertable via `create_hypertable()`; unique index `(device_id, f_cnt)` | `migrations/000001_*.up.sql` |
| AC-04 | `internal/config/config.go` uses env tags, no hardcoded defaults for sensitive fields | `internal/config/config.go` |
| AC-05 | `Makefile` has targets: `run`, `build`, `migrate`, `test`, `test-race`, `lint` | `Makefile` |
| AC-06 | `README.md` contains: quick start steps, config reference table, make commands | `README.md` |

---

## Step 5 — IoT Specific Checks

For any task touching telemetry ingestion, always verify:

```bash
# Run targeted tests for IoT scenarios
go test ./internal/... -run TestTelemetry -v
go test ./internal/... -run TestValidator -v
go test ./internal/... -run TestDuplicate -v
```

Check manually:
- [ ] `TelemetryPayload` struct has: `rssi`, `snr`, `f_cnt`, `spreading_factor`, `sample_count`
- [ ] Unique constraint `(device_id, f_cnt)` exists in migration
- [ ] `battery_level` validation: `0 ≤ value ≤ 100` enforced
- [ ] Negative `raw_weight` rejected with `ValidationError`

---

## Step 6 — Make Your Decision

### VERIFIED ✅

All gates pass + all ACs individually verified:

```bash
# Update sprint file — change Status header
> **Status:** 🏆 VERIFIED

# Add Status History row
| YYYY-MM-DD | IN_REVIEW | VERIFIED | QA | All ACs verified. All quality gates pass. |
```

```bash
# Update task_registry.md — change status to VERIFIED
# Then commit
git add docs/sprints/
git commit -m "qa(INV-SPR01-TASK-001): VERIFIED — all ACs pass, coverage ≥ 80%, no races"
```

### REJECTED ❌

Any gate fails or any AC not implemented:

```bash
# Update sprint file — change Status header
> **Status:** ❌ REJECTED

# Add Status History row with reason
| YYYY-MM-DD | IN_REVIEW | REJECTED | QA | AC-03: missing f_cnt index. Coverage 62% < 80% |
```

Write a detailed **Rejection Report** in the sprint file Notes section:

```markdown
### QA Rejection Report — INV-SPR01-TASK-001

**Verified ACs:** AC-01 ✅, AC-02 ✅
**Failed ACs:**
- AC-03 ❌: `create_hypertable()` call missing in migration
- AC-05 ❌: `make test-race` target missing from Makefile

**Quality Gate Results:**
- Build: ✅
- go vet: ✅
- Tests: ✅
- Race detector: ❌ DATA RACE in config/config.go:45
- Coverage: ❌ 62% < 80%

**Required fixes:**
1. Add `create_hypertable('raw_telemetry', 'received_at')` to migration
2. Add `test-race` target to Makefile
3. Fix DATA RACE in config loader
```

```bash
# Commit the rejection
git add docs/sprints/
git commit -m "qa(INV-SPR01-TASK-001): REJECTED — see rejection report in sprint file"
```

---

## Step 7 — After Decision

### After VERIFIED
- Update `task_registry.md` → `🏆 VERIFIED`
- Notify Lead to close the task (`🔒 CLOSED`)
- Move to the next `👀 IN_REVIEW` task

### After REJECTED
- Developer reads the rejection report
- Developer fixes all issues, re-submits → `👀 IN_REVIEW` (new history row)
- QA repeats from Step 1

---

## Sprint 1 — QA Checklist Quick Reference

```
Task in review: INV-SPR01-TASK-001 — Setup Infrastructure

Required files to verify:
  ✓ go.mod                                         (AC-01)
  ✓ cmd/server/main.go                             (AC-01)
  ✓ internal/config/config.go                      (AC-04)
  ✓ docker-compose.yml                             (AC-02)
  ✓ config/mosquitto/mosquitto.conf                (AC-02)
  ✓ migrations/000001_create_raw_telemetry.up.sql  (AC-03)
  ✓ migrations/000001_create_raw_telemetry.down.sql (AC-03)
  ✓ Makefile                                       (AC-05)
  ✓ .env.example                                   (AC-04, AC-06)
  ✓ README.md                                      (AC-06)

Commands to run:
  go build ./...
  go vet ./...
  go test ./... -count=1
  go test -race -count=1 ./...
  go test ./... -cover
```

---

## Important File Paths

| File | When to use |
|------|-------------|
| `docs/sprints/task_registry.md` | Find IN_REVIEW tasks, update status |
| `docs/sprints/sprint_0N_*.md` | Read ACs, write VERIFIED/REJECTED + history row |
| `.agents/rules/golang-tester-rules.md` | Full QA standards reference |
| `.agents/rules/golang-developer-rules.md` | Developer standards you enforce |

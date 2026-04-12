# BA Task Creation & State Tracking Workflow

> **Version:** 1.0  
> **Reference Standards:** IEC 62304, FDA 21 CFR Part 11, ISO 13485  
> **Project:** Inventory Management System (IoT Scale)  
> **Approver:** Project Lead / QA Lead  

---

## 1. Purpose

This document defines the standard process for the **Business Analyst (BA)** to create, track, and close tasks in each sprint, ensuring:
- Full traceability from requirement → implementation → testing
- Naming convention following FDA document control standards
- Audit trail for every status change

---

## 2. Task ID — FDA Naming Convention

### 2.1 Identifier Syntax

```
INV-SPR[NN]-[TYPE]-[SEQ]
```

| Part      | Meaning                         | Example    |
|-----------|---------------------------------|------------|
| `INV`     | Project code (Inventory)        | INV        |
| `SPR[NN]` | Sprint number, 2 digits         | SPR01      |
| `[TYPE]`  | Task type (see table below)     | TASK       |
| `[SEQ]`   | 3-digit zero-padded sequence    | 001        |

**Valid examples:**
```
INV-SPR01-TASK-001   ← Standard task
INV-SPR02-REQ-001    ← Requirement
INV-SPR03-TEST-002   ← Test case
INV-SPR04-BUG-001    ← Bug / Defect
INV-SPR05-CR-001     ← Change Request
```

### 2.2 Task Type Table (TYPE)

| TYPE   | Description                       | When to use                           |
|--------|-----------------------------------|---------------------------------------|
| `TASK` | Standard implementation task       | New feature, component                |
| `REQ`  | Functional Requirement            | When tracing from US/SRS              |
| `TEST` | Test Specification / Test Case    | Test plan, E2E scenario               |
| `BUG`  | Defect / discovered bug           | During testing or production          |
| `CR`   | Change Request (scope change)     | When requirements change post-APPROVED|

### 2.3 SEQ Numbering Rules

- SEQ starts at `001` within each sprint and increments continuously
- SEQ **must never be reused** even if a task is cancelled/voided
- Cancelled task: update status → `VOID`, do not delete the record
- Example: If `INV-SPR01-TASK-003` is voided, the next task is still `INV-SPR01-TASK-004`

---

## 3. Task Status Lifecycle (State Machine)

### 3.1 State Diagram

```
                    ┌─────────┐
                    │  DRAFT  │ 📝  ← BA creates task
                    └────┬────┘
                         │ BA submits for review
                         ▼
               ┌──────────────────┐
               │  PENDING_REVIEW  │ 🔍  ← Awaiting Lead/QA approval
               └────────┬─────────┘
              ┌─────────┴──────────┐
              │ Approved           │ Rejected
              ▼                    ▼
        ┌──────────┐         ┌──────────┐
        │ APPROVED │ ✅      │ REJECTED │ ❌ → (BA revises → DRAFT)
        └─────┬────┘         └──────────┘
              │ Dev picks up task
              ▼
        ┌─────────────┐
        │ IN_PROGRESS │ 🔄  ← Being implemented
        └──────┬──────┘
               │ Dev submits PR
               ▼
         ┌───────────┐
         │ IN_REVIEW │ 👀  ← Code / PR under review
         └─────┬─────┘
          ┌────┴────┐
          │ Pass    │ Fail
          ▼         ▼
     ┌──────────┐  IN_PROGRESS (rework)
     │ VERIFIED │ 🏆  ← Tests pass, QA sign-off
     └─────┬────┘
           │ Lead closes task
           ▼
       ┌────────┐
       │ CLOSED │ 🔒  ← Completed and archived
       └────────┘

   Any state (except CLOSED) → BLOCKED ⛔ (if a dependency blocks progress)
   BLOCKED → previous state (once unblocked)
```

### 3.2 Full Status Table

| Status Code      | Icon | Meaning                           | Owner                  |
|------------------|------|-----------------------------------|------------------------|
| `DRAFT`          | 📝   | Task is being drafted             | BA                     |
| `PENDING_REVIEW` | 🔍   | Awaiting Lead/QA review           | Lead / QA Lead         |
| `APPROVED`       | ✅   | Approved, ready to enter sprint   | Lead                   |
| `IN_PROGRESS`    | 🔄   | Developer is implementing         | Developer              |
| `IN_REVIEW`      | 👀   | PR is under review                | Reviewer               |
| `VERIFIED`       | 🏆   | Tests passed, QA verified         | QA                     |
| `CLOSED`         | 🔒   | Completed and officially closed   | Lead                   |
| `REJECTED`       | ❌   | Rejected (needs revision)         | —                      |
| `BLOCKED`        | ⛔   | Blocked by a dependency           | —                      |
| `VOID`           | 🚫   | Cancelled (append-only record)    | Lead                   |

---

## 4. BA Workflow — Creating a New Task

### Step 1 — Identify the Sprint and Domain

```
New requirement
    │
    ├─ Infrastructure / DevOps / Config?     → Sprint 1
    ├─ Device Management / Calibration?      → Sprint 2
    ├─ Calculation / Business Rules?         → Sprint 3
    ├─ Notifications / ERP / Automation?     → Sprint 4
    └─ Optimization / Fail-safe / E2E Test?  → Sprint 5
```

### Step 2 — Get the Next Task ID

1. Open `docs/sprints/task_registry.md`
2. Find the relevant sprint, identify the highest SEQ
3. Increment by 1: `max_seq + 1` (zero-padded to 3 digits)
4. Verify no collision with existing IDs (including VOID tasks)

### Step 3 — Create the Task in the Sprint File

Open the corresponding sprint file and add a task block using the template below:

```markdown
---

## [INV-SPRnn-TYPE-SEQ] — <Task Name>

> **Task ID:** `INV-SPRnn-TYPE-SEQ`  
> **Status:** 📝 DRAFT  
> **Created by:** <BA Name>  
> **Created date:** YYYY-MM-DD  
> **Assignee:** —  
> **Sprint:** n  

**Description:**  
<Clear description of what needs to be done and what the output is>

**Acceptance Criteria:**
- [ ] AC-01: <Action verb> + <measurable outcome>
- [ ] AC-02: ...
- [ ] AC-03: ...

**Related Technologies:**  
- <Libraries / Patterns / Tools to be used>

**Notes / Dependencies:**  
- Depends on: <Other Task ID if applicable>

**Status History:**
| Date       | From        | To               | Performed by    | Notes                |
|------------|-------------|------------------|-----------------|----------------------|
| YYYY-MM-DD | —           | DRAFT            | <BA Name>       | Task created         |
```

### Step 4 — Register in Task Registry

Open `docs/sprints/task_registry.md` and add a new row to the corresponding sprint table:

```
| INV-SPRnn-TYPE-SEQ | <Task name> | 📝 DRAFT | YYYY-MM-DD | — |
```

### Step 5 — Submit for Review

1. Update the status in the sprint file to `🔍 PENDING_REVIEW`
2. Add a new row to the task's status history
3. Update `task_registry.md` (Status column)
4. Notify Lead/QA to review

---

## 5. Status Update Workflow

### Approving (APPROVED)

Owner: **Lead**

1. Open the sprint file → locate the task by ID
2. Change status header: `✅ APPROVED`
3. Add history row: `| YYYY-MM-DD | PENDING_REVIEW | APPROVED | Lead Name | Approval reason |`
4. Update `task_registry.md` → Status column

### Starting Work (IN_PROGRESS)

Owner: **Developer**

1. Update the `Assignee` field in the task
2. Change status: `🔄 IN_PROGRESS`
3. Add history row
4. Update Sprint Metadata → `🔄 In Progress`

### Submitting PR (IN_REVIEW)

Owner: **Developer**

1. Change status: `👀 IN_REVIEW`
2. Add PR link in the `Notes` field
3. Add history row

### QA Verification (VERIFIED)

Owner: **QA**

1. Tick `[x]` for all passing Acceptance Criteria
2. Change status: `🏆 VERIFIED`
3. Add history row with test evidence reference

### Closing the Task (CLOSED)

Owner: **Lead**

1. Change status: `🔒 CLOSED`
2. Add history row
3. Check the sprint's Definition of Done
4. If all tasks are CLOSED → update Sprint Metadata → `✅` and update `_overview.md`

---

## 6. Audit Trail Rules

> **Mandatory under FDA 21 CFR Part 11:** Every status change MUST be recorded.

- **Never delete** status history rows
- **Never edit** Acceptance Criteria after APPROVED (create a `CR` task instead)
- If scope needs to change: create `INV-SPRnn-CR-SEQ` and link it to the original task
- Every REJECTED must include a specific reason in the Notes column

---

## 7. BA Checklist Before Submitting for Review

```
[ ] Task ID correct format: INV-SPR[NN]-[TYPE]-[SEQ]
[ ] Task name is concise and accurately describes the output
[ ] Description is clear: WHAT (what to do) + WHY (why it's needed)
[ ] Each AC starts with an action verb and is measurable
[ ] Related technologies are filled in
[ ] Dependencies are documented if applicable
[ ] Added to task_registry.md
[ ] Status history has the first row (DRAFT)
```

---

## 8. File Relationship Diagram

```
.agents/rules/golang-ba.md        ← Rules & conventions (trigger: always_on)
         │
         ├── docs/workflows/
         │   └── ba_task_creation_workflow.md   ← (this file) — detailed process
         │
         └── docs/sprints/
             ├── _overview.md                   ← Overall progress table
             ├── task_registry.md               ← Master list of all task IDs
             ├── sprint_01_infrastructure_ingestion.md
             ├── sprint_02_device_calibration.md
             ├── sprint_03_inventory_rules_engine.md
             ├── sprint_04_action_erp_integration.md
             └── sprint_05_optimization_failsafe.md
```

---

*This document is part of the Document Control system for project INV. Version 1.0 — 2026-04-12*

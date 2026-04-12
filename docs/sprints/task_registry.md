# Task Registry — Inventory Management System

> **Master list of all tasks following FDA naming convention**  
> **Standards:** IEC 62304 | FDA 21 CFR Part 11  
> **Last updated:** 2026-04-12 (Rev 2 — customer PDF requirement update)

---

## Usage Guide

- **Before creating a new task:** Find `max(SEQ)` in the corresponding sprint → use `max + 1`
- **SEQ must never be reused** even if a task is VOID
- **Update this table immediately** after any status change in the sprint files

### Status Legend

| Icon | Status           |
|------|------------------|
| 📝   | DRAFT            |
| 🔍   | PENDING_REVIEW   |
| ✅   | APPROVED         |
| 🔄   | IN_PROGRESS      |
| 👀   | IN_REVIEW        |
| 🏆   | VERIFIED         |
| 🔒   | CLOSED           |
| ❌   | REJECTED         |
| ⛔   | BLOCKED          |
| 🚫   | VOID             |

---

## Sprint 1 — Infrastructure & Data Ingestion

| Task ID               | Task Name                              | Status          | Assignee  | Updated     |
|-----------------------|----------------------------------------|-----------------|-----------|-------------|
| INV-SPR01-TASK-001    | Setup Infrastructure                   | 🔒 CLOSED       | Developer | 2026-04-12  |
| INV-SPR01-TASK-002    | Gateway Message Receiver               | 🔒 CLOSED       | Developer | 2026-04-13  |
| INV-SPR01-TASK-003    | Telemetry Validator & Data Parser      | 🏆 VERIFIED     | Developer | 2026-04-13  |
| INV-SPR01-TASK-004    | Raw Storage                            | ✅ APPROVED     | —         | 2026-04-12  |

> **Amended tasks:** TASK-003 (AC-07/08/09 added), TASK-004 (AC-07/08 added) — customer PDF Rev 2

**Sprint 1 Status:** 🔄 In Progress | Tasks: 4 total / 2 done | TASK-003 up next

---

## Sprint 2 — Device Management & Calibration

| Task ID               | Task Name                              | Status     | Assignee | Updated     |
|-----------------------|----------------------------------------|------------|----------|-------------|
| INV-SPR02-TASK-001    | Device Registry                        | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR02-TASK-002    | Calibration Manager                    | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR02-TASK-003    | Calibration Workflow                   | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR02-TASK-004    | Audit Trail                            | 📝 DRAFT   | —        | 2026-04-12  |

**Sprint 2 Status:** 🔲 Not Started | Tasks: 4 total / 0 done

---

## Sprint 3 — Inventory Rules Engine

| Task ID               | Task Name                              | Status     | Assignee | Updated     |
|-----------------------|----------------------------------------|------------|----------|-------------|
| INV-SPR03-TASK-001    | Weight Conversion Logic                | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-002    | Inventory Calculation                  | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-003    | Threshold Rules                        | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-004    | Historical Reporting API               | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-005    | Stock-out / Day Zero Forecasting       | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-006    | Consumption Anomaly Detection          | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR03-TASK-007    | Inventory Dashboard API                | 📝 DRAFT   | —        | 2026-04-12  |

> **Amended tasks:** TASK-001 (AC-08 added) — customer PDF Rev 2  
> **New tasks:** TASK-005, TASK-006, TASK-007 — customer PDF Rev 2

**Sprint 3 Status:** 🔲 Not Started | Tasks: 7 total / 0 done

---

## Sprint 4 — Action & ERP Integration

| Task ID               | Task Name                              | Status     | Assignee | Updated     |
|-----------------------|----------------------------------------|------------|----------|-------------|
| INV-SPR04-TASK-001    | Alert & Notification Service           | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR04-TASK-002    | ERP Integration Service                | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR04-TASK-003    | Scheduled FTP Upload                   | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR04-TASK-004    | Reorder Workflow                       | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR04-TASK-005    | Node Management & Calibration UI       | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR04-TASK-006    | Alerts Center UI                       | 📝 DRAFT   | —        | 2026-04-12  |

> **Amended tasks:** TASK-001 (AC-08/09 added), TASK-002 (AC-08/09 added) — customer PDF Rev 2  
> **New tasks:** TASK-005, TASK-006 — customer PDF Rev 2 (Management UI)

**Sprint 4 Status:** 🔲 Not Started | Tasks: 6 total / 0 done

---

## Sprint 5 — Optimization & Fail-safe

| Task ID               | Task Name                              | Status     | Assignee | Updated     |
|-----------------------|----------------------------------------|------------|----------|-------------|
| INV-SPR05-TASK-001    | Power Strategy Monitoring              | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR05-TASK-002    | Error Handling                         | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR05-TASK-003    | Data Aggregation Optimization          | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR05-TASK-004    | Final MVP Integration E2E              | 📝 DRAFT   | —        | 2026-04-12  |
| INV-SPR05-TASK-005    | Analytics Hub UI                       | 📝 DRAFT   | —        | 2026-04-12  |

> **New tasks:** TASK-005 — customer PDF Rev 2 (Analytics Hub UI)

**Sprint 5 Status:** 🔲 Not Started | Tasks: 5 total / 0 done

---

## Summary

| Sprint   | Total | DRAFT | IN_PROGRESS | VERIFIED | CLOSED |
|----------|-------|-------|-------------|----------|--------|
| Sprint 1 | 4     | 2     | 0           | 0        | 2      |
| Sprint 2 | 4     | 4     | 0           | 0        | 0      |
| Sprint 3 | 7     | 7     | 0           | 0        | 0      |
| Sprint 4 | 6     | 6     | 0           | 0        | 0      |
| Sprint 5 | 5     | 5     | 0           | 0        | 0      |
| **Total**| **26**| **26**| **0**       | **0**    | **0**  |

---

## Change Log

| Rev | Date       | Changed by | Description                                                      |
|-----|------------|------------|------------------------------------------------------------------|
| 1   | 2026-04-12 | BA         | Initial registry — 20 tasks across 5 sprints                     |
| 2   | 2026-04-12 | BA         | +6 new tasks, +5 amended tasks based on customer PDF requirement |

---

*Managed according to workflow: `docs/workflows/ba_task_creation_workflow.md`*

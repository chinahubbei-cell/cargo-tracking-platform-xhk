# Development Allocation Logic v2 (coding-agent + tmux)

Date: 2026-03-04
Owner: main assistant

## Goal
Use `coding-agent` for execution and `tmux` for stable multi-session control, with hard anti-idle guarantees.

## 1) Task Packaging Standard (before dispatch)
Each task must include:
- Objective (one sentence)
- Scope (files allowed)
- DoD (commit + test + risk note)
- Timeout SLA (ack/evidence/finish)

Template:
- TaskID:
- Agent: claude|codex
- Files:
- API/SQL:
- DoD:
- SLA: ack<=60s, evidence<=120s, first-commit<=30m

## 2) Agent Role Split
- Claude: architecture-sensitive backend logic, auth/permission paths
- Codex: implementation-heavy edits, refactor, tests
- Frontend package: whichever agent has fewer active sessions

## 3) File Locking (no overlap)
- One package owns one file set.
- If overlap is required, split by function boundary or sequence, never parallel edit same file.

## 4) tmux Execution Model
- One tmux session per package:
  - pkg-backend-auth
  - pkg-device-service
  - pkg-frontend-ui
- Keep session logs capturable for audit:
  - capture-pane output is the evidence source.

## 5) Hard Anti-Idle Gates
- Gate A: Ack <= 60s
  - Missing -> auto-retry once
- Gate B: Evidence <= 120s (diff/plan/command output)
  - Missing -> auto-switch agent or local takeover
- Gate C: First commit <= 30m
  - Missing -> force-scope reduction + immediate deliverable

## 6) Progress Reporting Contract (every 30m)
Report only:
- done: commits/tests merged
- doing: active package + ETA
- blocked: concrete blocker + mitigation started
Do not report "dispatched" as progress.

## 7) Merge & Validation Gate
Before claiming completion:
1. SQL migration order check
2. API error-code compatibility check
3. Minimal E2E smoke check
4. Rollback note

## 8) Fallback Strategy
If ACP unstable:
- Keep tmux sessions as control plane
- Use coding-agent in smaller packages
- If still unstable, local direct implementation for critical path first

## 9) Immediate P0 Package Plan (this project)
- Pkg-A (Claude): org service login guard + ORG_SERVICE_DISABLED
- Pkg-B (Codex): device service expiry guard + DEVICE_SERVICE_EXPIRED + assign-sub-account API
- Pkg-C (Codex/Claude): frontend login disabled page + device expiry prompt

Definition of closure:
- 3 package commits + one integration commit + test evidence

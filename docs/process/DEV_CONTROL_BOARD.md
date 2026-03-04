# DEV_CONTROL_BOARD

Last Updated: 2026-03-04 14:10 (Asia/Shanghai)

## Rules
- Single source of truth for development execution status
- Status flow only: `todo -> running -> evidence -> committed -> tested -> done`
- Progress counts only when there is commit/test evidence
- SLA:
  - Ack <= 60s
  - Evidence <= 120s
  - First commit <= 30m
  - No commit 60m => force local takeover

## Live Metrics
- Throughput (last 30m): 0 commits
- Lead Time (running -> committed): N/A
- Blocked Time (today): accumulating

## Work Packages (P0)

| Task ID | Scope | Owner | Files (lock) | Status | Start | Latest Evidence | Commit | Next ETA | Blocker |
|---|---|---|---|---|---|---|---|---|---|
| P0-A | 组织服务登录拦截 + ORG_SERVICE_DISABLED | main-local | trackcard-server/middleware/*, handlers/auth* | running | 14:10 | pending | - | 14:30 | - |
| P0-B | 设备到期判定 + DEVICE_SERVICE_EXPIRED | pending | trackcard-server/handlers/device*, services/device* | todo | - | - | - | 15:00 | wait P0-A |
| P0-C | assign-sub-account API + 前端最小提示 | pending | handlers/device*, frontend login/devices pages | todo | - | - | - | 15:30 | wait P0-B |

## Evidence Links
- Run folder: `docs/process/runs/2026-03-04/`
- Design: `docs/plans/2026-03-04-service-device-integration-design.md`
- Task breakdown: `docs/plans/2026-03-04-implementation-task-breakdown-v1.md`
- Allocation logic v2: `docs/process/development-allocation-logic-v2.md`

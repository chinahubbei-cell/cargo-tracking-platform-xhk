# P0-A Task Card

## Objective
组织到期/停用时不可登录，并返回统一错误码 `ORG_SERVICE_DISABLED`。

## Scope (file lock)
- trackcard-server middleware/auth related files
- trackcard-server handlers/auth related files
- minimal frontend handling if needed (error code display only)

## DoD
1. Backend login/auth path blocks suspended/expired org
2. Returns code/message with `ORG_SERVICE_DISABLED`
3. Minimal verification evidence recorded
4. Commit produced

## SLA
- ack <= 60s
- evidence <= 120s
- first commit <= 30m

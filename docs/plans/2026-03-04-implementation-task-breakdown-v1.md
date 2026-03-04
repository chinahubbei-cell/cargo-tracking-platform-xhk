# 货物追踪平台 × 管理后台打通 - 实施任务拆分表 v1

日期：2026-03-04  
状态：可开工

## 0. 目标边界（本期）
- 组织到期/停用：不可登录，展示系统禁用
- 设备服务期独立：设备到期即禁用（与组织并行判定）
- 设备归属仅二级：组织 -> 子账号（复用 users）
- 管理后台设备页不展示运单绑定信息
- 归属同步：准实时（1 分钟增量 + 5 分钟对账）

---

## 1. 任务总览（按优先级）

| 优先级 | 模块 | 任务 | 输出 |
|---|---|---|---|
| P0 | 追踪平台后端 | 组织服务拦截 | 登录/鉴权拒绝 + 统一错误码 |
| P0 | 追踪平台后端 | 设备服务期限判定 | 到期禁用 + 设备侧错误码 |
| P0 | 追踪平台后端 | 设备归属二级字段与接口 | assign-sub-account API |
| P0 | 管理后台后端 | 组织服务控制增强 | 状态流转、续期、审计 |
| P1 | 管理后台后端 | 设备页筛选与续费 | 按 org/sub/service 查询与续费 |
| P1 | 同步任务 | 归属增量同步 + 对账修复 | 1-5 分钟收敛 |
| P1 | 前端 | 登录停用页 + 设备到期提示 | UI 可感知 |
| P2 | 监控 | 同步失败/延迟告警 | 可观测与重试 |

---

## 2. SQL 变更清单

## 2.1 trackcard-server（追踪侧）
```sql
-- 组织服务状态（如未存在）
ALTER TABLE organizations ADD COLUMN service_status TEXT DEFAULT 'active';
ALTER TABLE organizations ADD COLUMN service_start DATETIME;
ALTER TABLE organizations ADD COLUMN service_end DATETIME;
CREATE INDEX IF NOT EXISTS idx_org_service_status_end ON organizations(service_status, service_end);

-- 设备服务期限
ALTER TABLE devices ADD COLUMN service_start_at DATETIME;
ALTER TABLE devices ADD COLUMN service_end_at DATETIME;
ALTER TABLE devices ADD COLUMN service_status TEXT DEFAULT 'active';
ALTER TABLE devices ADD COLUMN service_source TEXT;
ALTER TABLE devices ADD COLUMN service_updated_at DATETIME;
CREATE INDEX IF NOT EXISTS idx_devices_service_end ON devices(service_end_at);
CREATE INDEX IF NOT EXISTS idx_devices_service_status ON devices(service_status);

-- 设备归属（子账号）
ALTER TABLE devices ADD COLUMN sub_account_id TEXT;
CREATE INDEX IF NOT EXISTS idx_devices_org_sub ON devices(org_id, sub_account_id);
```

## 2.2 同步任务基础表
```sql
CREATE TABLE IF NOT EXISTS sync_cursors (
  id TEXT PRIMARY KEY,
  cursor_value TEXT,
  updated_at DATETIME
);

CREATE TABLE IF NOT EXISTS sync_failures (
  id TEXT PRIMARY KEY,
  biz_key TEXT,
  payload TEXT,
  error TEXT,
  created_at DATETIME
);
```

---

## 3. 具体文件/API 拆分

## 3.1 追踪平台后端（trackcard-server）

### T1 组织服务拦截
- 文件：
  - `middleware/auth.go`（或现有鉴权中间件）
  - `handlers/auth.go`
  - `models/organization.go`
- 变更：
  - 登录与 token 校验时加载组织服务状态
  - `service_status in (suspended,expired)` 或 `service_end < now` -> 拒绝
- 错误码：`ORG_SERVICE_DISABLED`

### T2 设备服务期限
- 文件：
  - `models/device.go`
  - `handlers/device.go`
  - `services/device_service.go`（若无则新建）
- 变更：
  - 设备查询/轨迹接口前统一判定 `service_end_at`
  - 到期设备拒绝获取轨迹
- 错误码：`DEVICE_SERVICE_EXPIRED`

### T3 设备归属二级（组织->子账号）
- 文件：
  - `models/device.go`
  - `handlers/device.go`
  - `services/device_binding_service.go`
- API：
  - `PUT /api/devices/:id/assign-sub-account`
- 入参：`org_id`, `sub_account_id`
- 校验：
  - 子账号必须存在于 users
  - 子账号须属于目标组织

### T4（已完成）运输节点最新时间缺失修复
- 文件：
  - `services/shipment_stage.go`
  - `../trackcard-frontend/src/components/TransportNodeTimeline.tsx`
- 提交：`f02839b`

## 3.2 管理后台后端（trackcard-admin/admin-server）

### T5 组织服务控制增强
- 文件：
  - `handlers/organization.go`
  - `models/organization.go`
- API：
  - `PUT /api/admin/orgs/:id/service`
  - `POST /api/admin/orgs/:id/renew`
- 规则：
  - 允许 active/suspended/expired 流转
  - 续期自动刷新 `service_end`

### T6 设备续费与查询增强
- 文件：
  - `handlers/device.go`
  - `models/device.go`
- API：
  - `GET /api/admin/devices?org_id=&sub_account_id=&service_status=`
  - `POST /api/admin/devices/:id/renew`
- 约束：
  - 管理后台设备侧不展示运单绑定信息

## 3.3 同步与调度

### T7 归属增量同步
- 文件：
  - `trackcard-server/services/sync_device_allocation.go`（新增）
  - `trackcard-server/services/sync_scheduler.go`
- 机制：
  - 每 1 分钟拉取归属变更日志增量
  - upsert 到后台设备归属视图

### T8 对账修复
- 文件：
  - `trackcard-server/services/sync_device_allocation.go`
- 机制：
  - 每 5 分钟比对 `device_id` 归属
  - 不一致自动修复并落 `sync_failures`

## 3.4 前端（trackcard-frontend）

### T9 登录停用页
- 文件：
  - `src/pages/Login.tsx`
  - `src/api/client.ts`
- 变更：
  - 识别 `ORG_SERVICE_DISABLED` 显示系统禁用文案

### T10 设备管理归属与到期提示
- 文件：
  - `src/pages/Devices.tsx`
  - `src/components/DeviceAssignModal.tsx`（新增）
- 变更：
  - 分配子账号
  - 设备到期显著提示

---

## 4. 验收用例（UAT）
1. 组织到期后登录失败，显示系统禁用
2. 组织到期但设备未到期：设备仍采集
3. 组织未到期但设备到期：仅该设备禁用
4. 设备续费后恢复可用
5. 归属变更 1-5 分钟同步到后台
6. 管理后台设备页不出现运单绑定信息
7. 运单 `260128000004` 最新运输节点时间可见

---

## 5. 今日最小开工闭环（建议）
- 先做：T1 + T2 + T3 + T9（形成“可登录控制 + 可分配 + 可提示”主闭环）
- 再做：T5 + T6（运营后台可控）
- 最后做：T7 + T8（一致性收敛）

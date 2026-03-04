# 货物追踪平台 × 管理后台打通设计（服务期限 + 设备归属）

日期：2026-03-04  
状态：已评审（可进入研发拆解）

---

## 1. 背景与目标

当前项目中，管理后台已具备组织服务字段（`organizations.service_status/service_start/service_end`）与硬件归属字段（`hardware_devices.org_id/sub_account_id`），但追踪平台与管理后台在“服务状态约束”和“设备归属同步”上未形成闭环。

本次目标：

1. 实现组织服务期限开启/关闭，并在追踪平台生效；
2. 实现设备归属二级模型（组织 → 子账号），子账号复用追踪平台 `users`；
3. 实现设备归属从追踪平台自动同步到管理后台（先准实时）；
4. 支持组织与设备分层到期策略，满足“账号停用 ≠ 设备停采”的业务规则。

---

## 2. 最终业务规则（已确认）

### 2.1 控制平面（账号）
- 判定依据：`organizations.service_end` + `service_status`
- 组织到期/停用后：
  - 不允许登录；
  - 登录页或鉴权后统一展示“系统禁用/账号已到期”；
  - 不提供历史轨迹只读入口；
  - 不允许新分配、续费提交等业务操作。

### 2.2 数据平面（设备）
- 判定依据：设备独立到期时间 `device_service_end`（命名见数据结构章节）。
- 设备到期前可继续采集并上报；
- 设备到期后，该设备禁用（禁止该设备轨迹获取/使用）；
- 设备续费后可恢复。

### 2.3 双层并行判定
- 账号是否可登录/可操作：看组织；
- 设备是否可用/可查轨迹：看设备；
- 两者互不覆盖。

---

## 3. 数据结构设计

## 3.1 复用与新增（管理后台）

已存在：
- `organizations`: `service_status`, `service_start`, `service_end`, `auto_renew`, `max_devices`;
- `hardware_devices`: `org_id`, `sub_account_id`, `allocated_at`, `status`;
- `service_renewals`, `device_allocation_logs`。

建议新增字段（`hardware_devices`）：
- `service_start_at DATETIME`：设备服务开始时间；
- `service_end_at DATETIME`：设备服务到期时间（设备级）；
- `service_status TEXT`：`active|expired|suspended`（设备级状态快照）；
- `service_source TEXT`：`order|auto_plus_1y|manual`（到期来源）；
- `service_updated_at DATETIME`。

说明：
- 设备到期来源规则：
  1) 优先出库订单到期时间；
  2) 无订单到期时，设备在追踪平台“添加设备激活时间 + 1年”。

## 3.2 追踪平台侧映射建议

在追踪平台设备域（`devices` 或关联表）保留：
- `org_id`（已存在）；
- `sub_account_id`（建议新增，关联 `users.id`）；
- `service_end_at`（设备级到期，必要）；
- `service_status`（可选快照字段，便于查询与筛选）。

子账号来源：直接复用追踪平台 `users`，不新建后台账号体系。

---

## 4. 核心流程设计

## 4.1 组织服务开关流程
1. 管理后台运营修改组织服务状态或到期时间；
2. 持久化到 `organizations`；
3. 追踪平台鉴权与业务中间件按组织状态拦截；
4. 客户登录时若组织不可用，统一停用提示页。

## 4.2 设备归属变更流程（追踪平台主导）
1. 客户在追踪平台将设备分配/转移给子账号；
2. 写入归属变更日志（含前后 org/sub_account）；
3. 同步任务将变更推送/拉取到管理后台 `hardware_devices`；
4. 管理后台更新归属与日志，形成运营可见视图。

## 4.3 准实时同步策略（已确认）
- 主链路：每 1 分钟增量同步；
- 补偿链路：每 5 分钟全量差异对账修复；
- SLA：95% 1 分钟内、100% 5 分钟内一致。

---

## 5. 状态机与优先级

## 5.1 组织状态机
`trial -> active -> expired`（按时间）  
`active|trial -> suspended`（手工）  
`suspended -> active`（恢复）

组织可登录条件：
- `service_status in (trial,active)` 且 `service_end >= now`（trial 可按产品策略是否需 end）。

## 5.2 设备状态机（服务维度）
`active -> expired`（到期）  
`expired -> active`（续费）  
`active -> suspended`（手工禁用，可选）

设备可用条件：
- `service_end_at >= now` 且 `service_status = active`。

---

## 6. 页面与交互设计（产品）

## 6.1 管理后台

### A. 组织详情页：服务控制台
- 展示：当前状态、开始/到期、自动续费、配额、影响设备数；
- 操作：立即开启、立即停用、续期N月、恢复服务；
- 停用前二次确认：展示影响（用户数、设备数、近24h活跃设备）。

### B. 设备管理页
- 新增筛选：组织、子账号、设备服务状态、设备到期时间区间；
- 列表新增：`sub_account_id/name`, `service_end_at`, `service_status`, `service_source`；
- 支持设备续费操作（批量/单个）。

## 6.2 追踪平台
- 登录页：组织到期/停用统一停用提示；
- 设备管理：支持分配给子账号（users）；
- 设备详情/轨迹页：设备到期时提示“设备已到期，请续费后恢复”。

---

## 7. API 设计（草案）

## 7.1 管理后台 API（新增/增强）
- `PUT /api/admin/orgs/:id/service`（已有，增强校验与审计）；
- `POST /api/admin/devices/:id/renew`（新增，设备续费）；
- `GET /api/admin/devices?service_status=&org_id=&sub_account_id=`（增强筛选）；
- `POST /api/admin/sync/device-allocations`（内部同步入口，可选）。

## 7.2 追踪平台 API（新增/增强）
- `PUT /api/devices/:id/assign-sub-account`（设备分配给子账号）；
- `GET /api/devices/:id/service-status`（设备服务判定）；
- 鉴权中间件统一返回：`ORG_SERVICE_DISABLED`；
- 设备查询统一返回：`DEVICE_SERVICE_EXPIRED`。

错误码建议：
- `ORG_SERVICE_DISABLED`（组织不可用）；
- `ORG_SERVICE_EXPIRED`（组织到期）；
- `DEVICE_SERVICE_EXPIRED`（设备到期）；
- `DEVICE_ASSIGN_FORBIDDEN`（组织不可用时禁止分配）。

---

## 8. 权限矩阵

- 客户用户：
  - 组织可用时可登录与常规操作；
  - 组织不可用时不可登录；
  - 设备到期时仅该设备受限。

- 运营后台管理员：
  - 可查看与修改组织服务；
  - 可查看设备归属与设备服务；
  - 可执行设备续费与组织恢复。

---

## 9. 非功能与风控

- 同步可观测：同步延迟、失败重试、对账差异率；
- 审计日志：组织服务改动、设备续费、设备归属变更全留痕；
- 幂等设计：同步按 `device_id + updated_at/version` 去重；
- 误操作防护：组织停用“高影响确认弹窗 + 可快速恢复”。

---

## 10. 验收标准（UAT）

1. 组织到期后客户无法登录，展示停用页；
2. 组织到期但设备未到期：设备仍可采集；
3. 组织未到期但设备已到期：仅该设备禁用且轨迹受限；
4. 设备续费后状态自动恢复为可用；
5. 追踪平台设备归属变更在 1-5 分钟内同步到管理后台；
6. 对账任务可修复归属不一致；
7. 所有关键动作有审计日志可追溯。

---

## 11. 分期建议

- **Phase 1（本期）**：组织服务开关、设备独立到期、组织→子账号归属、准实时同步、基础续费。  
- **Phase 2**：更精细化自动续费策略、账单对账、事件驱动实时同步（秒级）。

---

## 12. 决策记录（本次会话确认）

- 服务关闭策略：组织到期后不允许登录，展示系统禁用；
- 设备归属层级：仅二级（组织→子账号）；
- 子账号体系：复用追踪平台 `users`；
- 同步策略：先准实时；
- 期限判定：组织与设备双层并行，互不覆盖。

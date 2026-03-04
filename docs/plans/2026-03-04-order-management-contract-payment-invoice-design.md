# 管理后台订单管理模块产品设计（新购/续费）

日期：2026-03-04
状态：待确认（确认后可进入开发）

## 1. 目标
围绕硬件“新购”和“续费”订单，打通管理后台与货物追踪平台，形成完整交易闭环：
- 订单创建（手工 + 追踪平台提交）
- 审核流（统一待审核）
- 合同签约（在线/线下）
- 在线支付（先占位后对接真实支付）
- 开票（申请、审核、开票、回传）

## 2. 范围与约束
### In Scope
1) 管理后台订单管理模块（新购/续费）
2) 货物追踪平台“交易管理”菜单（设备购买、设备续费）
3) 订单审核、合同、支付、开票全链路状态机

### Out of Scope
1) 复杂财务对账系统
2) 多法币税率引擎
3) 全量合同模板中心（首版使用固定模板）

## 3. 核心业务决策（已确认）
1) 追踪平台提交订单后，进入**待审核**（不自动通过）
2) 订单类型：`purchase`（新购） / `renewal`（续费）
3) 合同签约支持：
   - 在线签约（平台内电子签）
   - 线下签约（上传已签扫描件）
4) 支付能力策略：选择 **C**（先占位后切真实）
   - Phase 1：在线支付占位 + 手工确认到账
   - Phase 2：对接真实支付通道（微信/支付宝/Stripe 等可配置）
5) 新购与续费订单均支持开票流程

## 4. 业务流程
## 4.1 新购订单
创建（后台手工/追踪平台） -> 待审核 -> 审核通过 -> 生成合同 -> 签约完成 -> 支付完成 -> 开票（可选） -> 履约（出库/分配） -> 完成

## 4.2 续费订单
创建（后台手工/追踪平台） -> 待审核 -> 审核通过 -> 生成合同 -> 签约完成 -> 支付完成 -> 开票（可选） -> 服务期延长 -> 完成

## 5. 状态机设计
### 5.1 订单状态（order_status）
- `draft`（草稿，仅后台手工可见）
- `pending_review`（待审核）
- `rejected`（驳回）
- `approved`（审核通过）
- `contract_pending`（待签约）
- `contract_signed`（已签约）
- `payment_pending`（待支付）
- `paid`（已支付）
- `invoice_pending`（待开票）
- `invoiced`（已开票）
- `fulfilling`（履约中）
- `completed`（完成）
- `cancelled`（取消）

### 5.2 合同状态（contract_status）
- `not_generated` / `generated` / `signing` / `signed_online` / `signed_offline` / `invalid`

### 5.3 支付状态（payment_status）
- `pending_payment`（待支付）
- `paid`（已支付）

联动规则：
- 订单支付状态与客户支付状态一一联动，仅保留“待支付/已支付”两态用于客户侧可见状态。
- 客户完成支付后，订单从 `payment_pending`（或等价待支付状态）流转到 `paid`。

### 5.4 开票状态（invoice_status）
- `not_requested` / `requested` / `approved` / `issued` / `delivered` / `rejected`

## 6. 功能设计
## 6.1 管理后台
### A. 订单管理菜单
- 列表：按订单类型（新购/续费）、来源（后台/追踪平台）、状态筛选
- 操作：审核、驳回、生成合同、确认签约、确认到账、开票处理、订单作废

订单作废规则：
- 仅在“未支付成功”前允许作废（即支付状态为待支付）。
- 一旦已支付，不允许直接作废，需走退款/冲正流程（后续阶段实现）。

### B. 审核中心
- 追踪平台提交订单默认进入此处
- 审核动作需要填写原因（通过/驳回）并留痕

### C. 合同管理
- 生成合同（模板渲染）
- 在线签约：发起签约、签约进度、签署结果回写
- 线下签约：上传签署文件、人工确认签约完成

### D. 支付管理
- Phase 1：生成支付单（占位），支持手工确认到账
- Phase 2：支付网关回调自动更新支付状态

### E. 开票管理
- 开票申请信息（抬头、税号、邮箱、地址）
- 审核开票申请
- 开票结果回填（发票号、开票时间、发票文件）

## 6.2 货物追踪平台
新增菜单：**交易管理**
- 子菜单1：设备购买
- 子菜单2：设备续费

功能：
- 创建订单（新购/续费）
- 查看订单状态（待审核/签约/支付/开票/完成）
- 合同签约入口（在线签约链接 / 线下上传）
- 支付入口（在线支付页）
- 开票申请入口

## 7. 数据模型（最小可落地）
### 7.1 orders
- id
- order_no
- order_type (`purchase`/`renewal`)
- source (`admin`/`tracking`)
- org_id
- sub_account_id (可空)
- amount_total
- currency
- order_status
- contract_status
- payment_status
- invoice_status
- reviewer_id/reviewed_at/review_comment
- created_by/created_at/updated_at

### 7.2 order_items
- id
- order_id
- item_type (`device`/`service_renewal`)
- device_id（续费可用）
- sku_id（新购可用）
- qty
- unit_price
- service_start_at/service_end_at（续费）

### 7.3 contracts
- id
- order_id
- contract_no
- sign_mode (`online`/`offline`)
- contract_status
- file_url
- signed_at

### 7.4 payments
- id
- order_id
- payment_no
- channel (`placeholder`/`wechat`/`alipay`/`stripe`...)
- amount
- payment_status
- paid_at
- raw_callback

### 7.5 invoices
- id
- order_id
- invoice_type
- title
- tax_no
- amount
- invoice_status
- invoice_no
- issued_at
- file_url

### 7.6 order_logs（审计）
- id
- order_id
- action
- from_status
- to_status
- operator_id
- note
- created_at

## 8. 权限与角色
- 超级管理员：全量可见、可审核、可取消、可开票
- 运营：审核、合同、支付确认、开票处理
- 财务：支付确认、开票处理
- 客户（追踪平台）：创建订单、签约、支付、申请开票、查看自身订单

## 9. 接口设计（摘要）
### 管理后台
- `POST /api/admin/orders`（手工创建）
- `GET /api/admin/orders`（列表）
- `POST /api/admin/orders/:id/review`（审核）
- `POST /api/admin/orders/:id/contract/generate`
- `POST /api/admin/orders/:id/contract/confirm-offline`
- `POST /api/admin/orders/:id/payment/confirm`
- `POST /api/admin/orders/:id/void`（未支付订单作废）
- `POST /api/admin/orders/:id/invoice/review`
- `POST /api/admin/orders/:id/invoice/issue`

### 追踪平台
- `POST /api/trade/orders/purchase`
- `POST /api/trade/orders/renewal`
- `GET /api/trade/orders`
- `GET /api/trade/orders/:id`
- `POST /api/trade/orders/:id/contract/sign-online`
- `POST /api/trade/orders/:id/contract/upload-offline`
- `POST /api/trade/orders/:id/payment/create`
- `POST /api/trade/orders/:id/invoice/apply`

## 10. 分期落地建议
### Phase 1（先上线）
- 订单创建（后台+追踪）
- 待审核流程
- 合同（在线入口+线下上传）
- 支付占位 + 手工确认到账
- 开票申请与后台处理

### Phase 2（增强）
- 真实支付通道接入
- 电子签深度集成
- 自动对账与发票回传

## 11. 验收标准（UAT）
1) 追踪平台新购/续费订单可提交并进入后台待审核
2) 管理后台可审核通过/驳回并留痕
3) 审核通过后可完成在线签约或线下签约
4) 客户支付状态与订单支付状态严格联动（待支付/已支付）
5) 支付后订单状态正确流转为已支付
6) 未支付订单可在后台作废，已支付订单不可作废
7) 开票申请可提交、审核、开票并回填结果
8) 续费订单完成后，设备/服务期限正确延长

## 12. 默认业务参数（已确认）
1) 审核通过后：**强制签约后才能支付**
2) 续费生效时点：**支付成功即生效**
3) 作废权限：**仅管理员可作废**
4) 支付超时：**待支付 24 小时自动失效**
5) 首期真实支付通道：**微信支付**（Phase 2）
6) 开票触发：**仅已支付订单可申请开票**
7) 发票首版范围：**先支持普票**

## 13. 权限矩阵（补充）
- 管理员：创建/审核/作废/确认到账/开票全权限
- 运营：创建/审核/合同处理/开票处理（不可作废）
- 财务：确认到账/开票处理
- 客户：提交订单、签约、支付、申请开票、查看自身订单

## 14. 异常回滚与补偿（补充）
1) 未支付订单作废：
- 新购：回滚库存预占
- 续费：若尚未生效则直接终止；若误生效则触发人工回滚工单
2) 支付超时失效：
- 自动流转为 `cancelled`（原因：payment_timeout）
3) 开票驳回：
- 允许客户重新提交开票申请
4) 回调幂等：
- 同一 payment_no 重复回调仅首次生效

## 15. 风险与控制
- 风险：状态流转复杂导致脏状态
  - 控制：状态机校验 + 日志审计
- 风险：支付回调异常
  - 控制：幂等处理 + 手工兜底确认
- 风险：线下签约真实性
  - 控制：强制上传签章文件 + 审核确认

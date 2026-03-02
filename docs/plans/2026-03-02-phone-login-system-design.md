# 货运追踪平台手机号登录体系设计方案（正式版）

- 日期：2026-03-02
- 项目路径：`/Users/tianxingjian/Aisoftware/cargo-tracking-platform-xhk`
- 适用范围：追踪系统（Track Frontend）+ 管理后台（Admin Frontend）
- 设计目标：新增手机号验证码登录、验证码重置密码、同手机号多机构切换

## 1. 背景与目标
当前系统以邮箱+密码为主。为降低登录门槛并统一账号体系，需要新增手机号认证能力，满足以下验收标准：
1. 手机号通过验证码直接登录系统。
2. 手机号通过验证码直接更换登录密码。
3. 同一个手机号可以归属多个机构，并可切换不同机构。

用户决策约束：
- 短信通道：首版采用中国大陆 `+86` 场景。
- 短信平台：首版真实接入 + 可插拔架构（A+C）。
- 机构选择：首次登录必须手动选择机构；后续默认上次使用机构。

## 2. 架构方案选择
采用**方案1（统一身份中心）**：在 `trackcard-server` 中统一实现手机号、验证码、机构选择、会话签发；追踪端与管理后台共同调用。

## 3. 登录流程设计
1) 用户输入手机号（默认 +86），请求验证码。
2) 服务端下发验证码并记录。
3) 用户提交手机号+验证码。
4) 验证通过后查询用户可访问机构列表。
5) 首次登录（无 last_org_id）必须手动选机构；非首次默认进入 last_org_id。
6) 签发 token（user_id + current_org_id + role_scope）。
7) 前端按 current_org_id 拉取权限与数据。

### 重置密码流程
- 验证码场景 `reset_password`，通过后更新 password_hash。

### 多机构切换流程
- 登录后可查询 org 列表，切换后重签 token，并更新 last_org_id。

## 4. 数据模型设计
### 4.1 users（扩展）
- phone_country_code（默认 +86）
- phone_number
- phone_verified_at
- last_org_id
- password_hash

约束：唯一索引 `(phone_country_code, phone_number)`。

### 4.2 user_org_memberships（复用或规范化）
- user_id, org_id, role, status, is_default(可选)

### 4.3 auth_verification_codes（新建）
- id, scene(login/reset_password), phone_country_code, phone_number
- code_hash, expires_at, used_at, request_ip, attempt_count, created_at

### 4.4 sms_send_logs（新建）
- provider, phone, template_code, biz_id, status, error_code, sent_at

## 5. API 设计（V1）
- `POST /api/auth/sms/send-code`
- `POST /api/auth/sms/login`
- `POST /api/auth/select-org`
- `POST /api/auth/password/reset-by-sms`
- `GET /api/auth/orgs`
- `POST /api/auth/switch-org`

## 6. 短信平台与可插拔设计
首版推荐：**阿里云短信服务（中国大陆）**。

抽象接口：`SmsProvider.SendCode(...)`
实现：
- AliyunSmsProvider（生产）
- MockSmsProvider（开发/测试）

## 7. 安全与风控
- 验证码 TTL 5 分钟
- 单手机号 60 秒/次，1小时上限 10 次
- 校验失败上限 5 次
- 验证码一次性使用，成功即失效
- 验证码只存 hash，日志不打明文

## 8. 验收映射
1. 手机验证码登录：send-code(login) + sms/login + select-org
2. 手机验证码重置密码：send-code(reset_password) + reset-by-sms
3. 同手机号多机构切换：memberships + orgs/switch-org + last_org_id

## 9. 风险与回滚
风险：历史账号手机号缺失、机构关系数据不规范、短信模板审核周期。
回滚：保留原邮箱密码登录入口 + feature flag 控制手机号登录开关。


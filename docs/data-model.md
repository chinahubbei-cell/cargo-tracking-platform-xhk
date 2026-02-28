# TrackCard 数据模型关系说明

## 概述

本文档描述 TrackCard 货运追踪平台的核心数据模型及其关系。

---

## 核心实体关系图

```
┌─────────────────────────────────────────────────────────────────────────┐
│                           Organization (组织机构)                         │
│  - 支持多级层级结构（集团 → 分公司 → 部门）                                    │
│  - 通过 parent_id 自引用实现树形结构                                         │
└────────────────┬───────────────────────────────────────┬────────────────┘
                 │                                       │
                 │ 1:N                                   │ 1:N
                 ▼                                       ▼
┌────────────────────────────┐           ┌────────────────────────────┐
│   UserOrganization (关联)   │           │      Shipment (运单)        │
│  - user_id                 │           │  - org_id (归属组织)         │
│  - organization_id         │           │  - device_id (绑定设备)      │
│  - is_primary (是否主部门)  │           │  - status, progress...     │
│  - position (职位)         │           └─────────────┬──────────────┘
└────────────┬───────────────┘                         │
             │                                         │ N:1
             │ N:1                                     ▼
             ▼                           ┌────────────────────────────┐
┌────────────────────────────┐           │       Device (设备)         │
│        User (用户)          │           │  - org_id (归属组织)         │
│  - role (admin/operator)   │           │  - external_device_id      │
│  - email, name, password   │           │  - latitude, longitude     │
└────────────────────────────┘           │  - status, battery...      │
                                         └────────────────────────────┘
```

---

## 实体详细说明

### 1. Organization (组织机构)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键，如 `org-hq` |
| name | string | 组织名称 |
| code | string | 组织编码 |
| parent_id | string? | 父组织ID，顶级组织为 null |
| type | string | 类型：group(集团)/branch(分公司)/dept(部门) |
| level | int | 层级深度，根节点为 1 |
| status | string | 状态：active/inactive |

**层级示例**：
```
快货运国际集团总部 (level=1, parent_id=null)
├── 华东分公司 (level=2, parent_id=org-hq)
│   ├── 运营部 (level=3, parent_id=org-east)
│   └── 技术部 (level=3, parent_id=org-east)
├── 华南分公司 (level=2, parent_id=org-hq)
└── 华中分公司 (level=2, parent_id=org-hq)
```

---

### 2. User (用户)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 主键，如 `user-xxx` |
| email | string | 登录邮箱（唯一） |
| name | string | 姓名 |
| password | string | 密码（bcrypt 加密） |
| role | string | 角色：admin/operator/viewer |
| status | string | 状态：active/disabled |

---

### 3. UserOrganization (用户-组织关联)

| 字段 | 类型 | 说明 |
|------|------|------|
| user_id | string | 用户ID |
| organization_id | string | 组织ID |
| is_primary | bool | 是否主部门 |
| position | string | 职位名称 |

**说明**：一个用户可以属于多个组织，但只有一个主部门。

---

### 4. Shipment (运单)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 运单号，格式 `YYMMDDXXXXXX` |
| device_id | string? | 绑定的设备ID |
| **org_id** | string? | **归属组织ID** |
| origin | string | 发货地 |
| destination | string | 目的地 |
| status | string | 状态：pending/in_transit/delivered/cancelled |
| progress | int | 进度百分比 0-100 |

**重要**：`org_id` 在创建时**自动设置**为当前用户的主组织。

---

### 5. Device (设备)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 设备ID，如 `XHK-001` |
| name | string | 设备名称 |
| external_device_id | string? | 第三方设备号 |
| **org_id** | string? | **归属组织ID** |
| status | string | 在线状态：online/offline |
| latitude | float64 | 纬度 |
| longitude | float64 | 经度 |
| battery | int | 电量百分比 |

**重要**：`org_id` 在创建时**自动设置**为当前用户的主组织。

---

## 数据权限规则

### 可见性规则

| 用户组织 | 可见数据范围 |
|----------|-------------|
| 属于根组织（总部） | 可见所有数据 |
| 属于分公司/部门 | 可见本组织及所有子组织的数据 |
| 无组织归属 | 仅可见无组织归属的数据 |

### 筛选逻辑

1. **不传 org_id**：按用户权限自动过滤
2. **传 org_id**：返回该组织及其子组织的数据（需有权限）
3. **传 org_id=''**：明确表示查看所有可见数据

---

## 新增实体时的必填字段

当新增类似运单、设备的业务实体时，需要包含：

```go
type NewEntity struct {
    ID        string   `gorm:"primaryKey"`
    OrgID     *string  `json:"org_id"`        // 必须：归属组织
    CreatedBy *string  `json:"created_by"`    // 建议：创建者
    // ... 其他字段
}
```

在 Create 处理器中：
```go
// 自动获取当前用户组织
var userOrgID *string
if userID, exists := c.Get("user_id"); exists {
    // ... 查询用户主组织
}
entity.OrgID = userOrgID
```

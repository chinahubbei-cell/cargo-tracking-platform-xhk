# TrackCard API 设计规范

## 概述

本文档定义 TrackCard 平台 API 的设计规范和最佳实践。

---

## 通用响应结构

### 成功响应

```json
{
  "success": true,
  "data": { ... }
}
```

### 列表响应

```json
{
  "success": true,
  "data": [...],
  "total": 100,
  "page": 1,
  "page_size": 20
}
```

### 错误响应

```json
{
  "success": false,
  "error": "错误信息",
  "code": "ERROR_CODE"
}
```

---

## URL 命名规范

| 操作 | HTTP 方法 | URL 模式 | 示例 |
|------|-----------|----------|------|
| 列表查询 | GET | `/api/{resources}` | `/api/shipments` |
| 获取单个 | GET | `/api/{resources}/{id}` | `/api/shipments/260118000001` |
| 创建 | POST | `/api/{resources}` | `/api/shipments` |
| 更新 | PUT | `/api/{resources}/{id}` | `/api/shipments/260118000001` |
| 删除 | DELETE | `/api/{resources}/{id}` | `/api/shipments/260118000001` |

---

## 关联字段返回规范

### ❌ 错误做法

只返回关联 ID，前端无法直接显示：
```json
{
  "id": "260118000001",
  "org_id": "org-east"
}
```

### ✅ 正确做法

同时返回 ID 和展示名称：
```json
{
  "id": "260118000001",
  "org_id": "org-east",
  "org_name": "华东分公司"
}
```

### 实现方式

在 List/Get 处理器中查询关联表：
```go
// 批量获取组织名称
var orgIDs []string
for _, item := range items {
    if item.OrgID != nil {
        orgIDs = append(orgIDs, *item.OrgID)
    }
}

var orgs []models.Organization
db.Where("id IN ?", orgIDs).Find(&orgs)
orgMap := make(map[string]string)
for _, org := range orgs {
    orgMap[org.ID] = org.Name
}

// 填充 org_name
for i := range result {
    if result[i].OrgID != nil {
        result[i].OrgName = orgMap[*result[i].OrgID]
    }
}
```

---

## 筛选参数规范

### 组织筛选

| 参数值 | 含义 |
|--------|------|
| 不传 `org_id` | 按用户权限自动过滤 |
| `org_id=org-east` | 查询该组织及其子组织 |
| `org_id=` (空字符串) | 明确表示查看全部可见数据 |

### 分页参数

| 参数 | 默认值 | 说明 |
|------|--------|------|
| page | 1 | 页码 |
| page_size | 20 | 每页条数，最大 100 |

### 搜索参数

| 参数 | 说明 |
|------|------|
| search | 模糊搜索关键词 |
| status | 状态筛选 |
| type | 类型筛选 |

---

## 创建接口规范

### 自动填充字段

创建实体时，以下字段应由后端自动填充：

| 字段 | 来源 |
|------|------|
| id | 自动生成（UUID 或业务规则） |
| org_id | 当前用户的主组织 |
| created_at | 当前时间 |
| created_by | 当前用户 ID（如适用） |

### 请求结构

使用指针类型区分"未传"和"传空值"：
```go
type CreateRequest struct {
    Name     string   `json:"name" binding:"required"`
    DeviceID *string  `json:"device_id"`  // 可选
    OrgID    *string  `json:"org_id"`     // 可选，不传则自动设置
}
```

---

## 更新接口规范

### 部分更新

使用指针类型，只更新传入的字段：
```go
type UpdateRequest struct {
    Name   *string `json:"name"`
    Status *string `json:"status"`
    OrgID  *string `json:"org_id"`
}

// 处理器中
updates := make(map[string]interface{})
if req.Name != nil {
    updates["name"] = *req.Name
}
if req.OrgID != nil {
    updates["org_id"] = *req.OrgID
}
db.Model(&entity).Updates(updates)
```

---

## 认证与授权

### 请求头

```
Authorization: Bearer <token>
Content-Type: application/json
```

### Token 结构

JWT 包含：
- `user_id`: 用户 ID
- `email`: 用户邮箱
- `role`: 用户角色
- `exp`: 过期时间

### 权限检查

在中间件中提取用户信息：
```go
c.Set("user_id", claims.UserID)
c.Set("user_role", claims.Role)
```

在处理器中使用：
```go
userID, _ := c.Get("user_id")
userRole, _ := c.Get("user_role")
```

---

## 错误处理

### 标准错误码

| HTTP 状态码 | 场景 |
|------------|------|
| 400 | 请求参数错误 |
| 401 | 未登录或 Token 失效 |
| 403 | 无权限访问 |
| 404 | 资源不存在 |
| 500 | 服务器内部错误 |

### 工具函数

```go
utils.BadRequest(c, "请提供有效的参数")
utils.Unauthorized(c, "请先登录")
utils.Forbidden(c, "无权限访问")
utils.NotFound(c, "资源不存在")
utils.InternalError(c, "服务器错误")
```

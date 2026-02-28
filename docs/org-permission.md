# TrackCard 组织权限体系说明

## 概述

本文档说明 TrackCard 平台的多组织权限体系设计和实现细节。

---

## 组织层级结构

### 层级类型

| 类型 | 说明 | 示例 |
|------|------|------|
| group | 集团/总部 | 快货运国际集团总部 |
| company | 子公司 | 华中分公司 |
| branch | 分公司/分支 | 华东分公司 |
| dept | 部门 | 运营部、技术部 |

### 典型组织结构

```
快货运国际集团总部 (group, level=1)
├── 华东分公司 (branch, level=2)
│   ├── 运营部 (dept, level=3)
│   └── 技术部 (dept, level=3)
├── 华南分公司 (branch, level=2)
└── 华中分公司 (company, level=2)
    └── 襄阳子公司 (branch, level=3)
```

---

## 用户与组织的关系

### 多组织归属

- 一个用户可以属于**多个组织**
- 必须指定一个**主部门**（is_primary=true）
- 每个归属可设置**职位**（position）

### 数据表：user_organizations

| 字段 | 说明 |
|------|------|
| user_id | 用户 ID |
| organization_id | 组织 ID |
| is_primary | 是否主部门 |
| position | 职位名称 |

---

## 数据权限规则

### 可见性矩阵

| 用户所属组织 | 可见数据范围 |
|-------------|-------------|
| 总部（根组织） | 所有数据 |
| 分公司 | 本分公司 + 所有子部门 |
| 部门 | 仅本部门 |
| 无组织 | 仅无归属数据 |

### 判断逻辑

```go
// 1. 检查是否属于根组织（可见全部）
func CanAccessAllData(userID string) bool {
    orgIDs := GetUserOrgIDs(userID)
    for _, orgID := range orgIDs {
        if IsRootOrg(orgID) {
            return true
        }
    }
    return false
}

// 2. 获取可见组织列表（包含子组织）
func GetVisibleOrgIDs(userID string) []string {
    userOrgIDs := GetUserOrgIDs(userID)
    visibleIDs := make(map[string]bool)
    
    for _, orgID := range userOrgIDs {
        visibleIDs[orgID] = true
        // 递归获取所有子组织
        for _, descendantID := range GetDescendantIDs(orgID) {
            visibleIDs[descendantID] = true
        }
    }
    return mapKeys(visibleIDs)
}
```

---

## 实现细节

### 1. DataPermissionService

位置：`trackcard-server/services/data_permission.go`

核心方法：
- `GetUserOrgIDs(userID)` - 获取用户所属组织
- `GetVisibleOrgIDs(userID)` - 获取可见组织（含子组织）
- `CanAccessAllData(userID)` - 是否可访问全部数据
- `FilterQueryByOrg(query, userID, orgField)` - 按权限过滤查询
- `ApplyOrgFilter(query, userID, requestedOrgID)` - 应用筛选条件

### 2. 在处理器中使用

```go
func (h *Handler) List(c *gin.Context) {
    userID := c.GetString("user_id")
    requestedOrgID := c.Query("org_id")
    
    query := h.db.Model(&Entity{})
    
    // 应用组织筛选
    query = services.DataPermission.ApplyOrgFilter(
        query, 
        userID, 
        requestedOrgID, 
        "org_id",  // 实体中的组织字段名
    )
    
    var items []Entity
    query.Find(&items)
}
```

### 3. 筛选参数处理

| 前端传参 | 后端行为 |
|----------|----------|
| 不传 org_id | 按用户权限自动过滤 |
| org_id=org-east | 查询该组织及子组织 |
| org_id= (空) | 查看全部可见数据 |

---

## 前端实现

### 1. 组织切换

用户可在右上角切换当前操作的组织：
- 存储在 `authStore.currentOrgId`
- 保存到 localStorage 持久化
- 切换后刷新页面应用新数据

### 2. 组织筛选下拉框

```tsx
// 展平组织树
const flattenOrgs = (orgs, level = 0) => {
    const result = [];
    for (const org of orgs) {
        result.push({ id: org.id, name: org.name, level });
        if (org.children?.length > 0) {
            result.push(...flattenOrgs(org.children, level + 1));
        }
    }
    return result;
};

// 下拉选项
<Select onChange={(v) => setFilterOrgId(v === 'all' ? '' : v)}>
    <Option value="all">全部组织</Option>
    {flatOrgs.map(org => (
        <Option key={org.id} value={org.id}>
            {'└ '.repeat(org.level)}{org.name}
        </Option>
    ))}
</Select>
```

### 3. API 请求自动附加组织

在 API 客户端拦截器中：
```typescript
// 只有当请求没有明确指定 org_id 时才自动添加
if (currentOrgId && !('org_id' in config.params)) {
    config.params = { ...config.params, org_id: currentOrgId };
}
```

---

## 常见问题

### Q: 为什么总部账号看不到子公司数据？

检查：
1. API 请求是否自动附加了 `org_id=org-hq`
2. 如果附加了，后端应该返回总部 + 所有子组织的数据
3. 如果想看全部，应该不传 org_id 或传空字符串

### Q: 新建数据没有组织归属？

原因：创建时未自动关联组织

解决：在 Create 处理器中添加：
```go
var userOrgID *string
if userID, exists := c.Get("user_id"); exists {
    // 查询用户主组织
    userOrgID = &userOrg.OrganizationID
}
entity.OrgID = userOrgID
```

### Q: 历史数据无组织归属怎么办？

方案：批量更新到默认组织
```bash
curl -X PUT "/api/shipments/{id}" -d '{"org_id":"org-hq"}'
```

# TrackCard 管理后台

## 项目结构

```
trackcard-admin/
├── admin-server/        # Go后端 (端口: 8001)
│   ├── handlers/        # API处理器
│   ├── models/          # 数据模型
│   ├── middleware/      # 中间件
│   └── main.go
├── admin-frontend/      # React前端 (端口: 5175)
└── docker-compose.yml
```

## 快速开始

### 启动后端
```bash
cd admin-server
go run main.go
```

### 启动前端
```bash
cd admin-frontend
npm run dev
```

## 默认账号

- 用户名: `admin`
- 密码: `admin123`

## API端点

| 模块 | 端点 | 描述 |
|-----|------|------|
| 认证 | POST /api/admin/auth/login | 管理员登录 |
| 组织 | GET/POST /api/admin/orgs | 组织管理 |
| 订单 | GET/POST /api/admin/orders | 订单管理 |
| 设备 | GET/POST /api/admin/devices | 设备管理 |
| 仪表盘 | GET /api/admin/dashboard | 统计数据 |

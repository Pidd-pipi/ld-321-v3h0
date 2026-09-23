# AgriDispatch（农机调度管理系统）

> 项目类型：全栈Web应用

面向农业合作社与种植大户的农机作业调度平台，支持农机资源管理、作业任务指派（推荐空闲农机/驾驶员）、实时地图轨迹监控（WebSocket + 高德地图）、作业记录统计报表、维修保养提醒与驾驶员管理。

## 快速启动（Docker Compose 一键部署，首选）

```bash
# 1. 首次启动前复制环境变量
cp .env.example .env

# 2. 启动全部服务（MySQL + Redis + 后端 + 前端）
docker compose up -d

# 3. 查看健康状态
docker compose ps
```

访问地址：

- 前端调度看板：http://localhost:18621
- 后端 API：http://localhost:19621
- 健康检查：http://localhost:19621/healthz
- Swagger：http://localhost:19621/docs/index.html

演示账号：`admin / admin123`

## 本地开发

```bash
# 前端（Vue 3 + TS + Vite + Element Plus + ECharts）
cd frontend
npm install
npm run dev        # http://localhost:18621

# 后端（Go）
cd backend
go mod tidy
go run ./cmd/server
```

## 技术栈

| 层次 | 技术 |
| --- | --- |
| 前端 | Vue 3 + TypeScript + Vite + Element Plus + ECharts |
| 地图 | 高德地图 API（JSAPI） |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7（go-redis/v9，看板缓存 + 实时推送） |
| 实时通信 | WebSocket（gorilla/websocket，农机实时位置推送） |
| 认证 | JWT (github.com/golang-jwt/jwt/v5) + RBAC |
| 配置 | github.com/caarlos0/env/v11 |
| 日志 | Go 标准库 log/slog |

## 项目目录结构

```
.
├── docker-compose.yml          # db + redis + backend + frontend
├── .env / .env.example
├── README.md
├── frontend/
│   ├── Dockerfile / nginx.conf # /api/ → backend:8080/api/v1/，/ws → backend:8080/ws
│   └── src/
│       ├── features/           # DashboardView
│       ├── components/         # MachineTable/TaskBoard/MapTrackPanel/...
│       ├── services/           # API 调用（overview/dispatch）
│       ├── stores/  types/  constants/  logger/  errors/
└── backend/
    ├── Dockerfile
    ├── database/init.sql
    ├── cmd/server/main.go
    └── internal/
        ├── config/  model/  repository/  service/  handler/
        ├── router/  middleware/  ws/          # WebSocket Hub
        ├── constants/  errors/  logger/  util/  database/
```

## 主要 API 列表

统一前缀 `/api/v1`，统一响应 `{ "code": 0, "message": "ok", "data": ... }`。

| 方法 | 路径 | 说明 | 鉴权 |
| --- | --- | --- | --- |
| POST | /auth/login | 登录 | - |
| GET | /auth/me | 当前用户 | JWT |
| GET | /dashboard/overview | 调度看板总览（农机/任务/轨迹/统计/保养/驾驶员，任务含最近派单记录） | - |
| POST | /tasks/:id/dispatch | 派单确认（body 可传 machineCode/driverName/reason，未传沿用推荐资源） | - |
| POST | /tasks/:id/reassign | 已派单任务改派（body 传 machineCode/driverName/reason，reason 必填） | - |
| GET | /dashboard/reports/work/export | 作业报表导出信息 | - |
| GET | /ws | WebSocket 实时轨迹推送 | - |
| GET | /healthz | 健康检查（DB + Redis） | - |

派单/改派在单个数据库事务内完成：任务须为待派单/已派单、目标农机须空闲、目标驾驶员须在岗或可派单；条件不满足返回 HTTP 409（错误码 40900）与明确中文原因，任务与资源状态均不变。成功后同步更新任务、农机、驾驶员三方状态并写入派单/改派记录（含原因与旧资源快照）。

派单请求体示例：

```json
{ "machineCode": "NJ-2026-002", "driverName": "何燕", "reason": "按地块就近派单" }
```

改派请求体示例：

```json
{ "machineCode": "NJ-2026-005", "driverName": "周明", "reason": "原农机进入保养" }
```

## 环境变量

| 变量 | 说明 | 默认值 |
| --- | --- | --- |
| COMPOSE_PROJECT_NAME | Compose 项目名 | agridispatch |
| APP_ENV | 运行环境 | development |
| SERVER_PORT | 后端端口（容器内） | 8080 |
| DB_HOST / DB_PORT | 数据库地址/端口 | db / 3306 |
| DB_NAME / DB_USER / DB_PASSWORD | 数据库配置 | agridispatch |
| DB_ROOT_PASSWORD | MySQL root 密码 | agridispatch_root_pwd |
| REDIS_HOST / REDIS_PORT | Redis 地址/端口 | redis / 6379 |
| REDIS_PASSWORD | Redis 密码 | 空 |
| JWT_SECRET | JWT 密钥 | 请修改 |
| FRONTEND_PORT | 前端宿主端口 | 18621 |
| BACKEND_PORT | 后端宿主端口 | 19621 |
| DB_PORT | MySQL 宿主端口 | 33321 |
| REDIS_PORT | Redis 宿主端口 | 36321 |

## Docker 部署说明

- 端口映射：前端 `18621:80`、后端 `19621:8080`、MySQL `33321:3306`、Redis `36321:6379`
- 数据卷：`db_data`、`redis_data`
- Nginx 反代：`/api/ → http://backend:8080/api/v1/`；`/ws` 配置 WebSocket Upgrade/Connection 头
- 常见问题：等待 `docker compose ps` 全部 healthy（MySQL 初始化约 30~120 秒）；高德地图 Key 在线上环境需按 `MapTrackPanel` 接入真实 JSAPI Key

## License

MIT License

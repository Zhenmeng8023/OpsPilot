# OpsPilot

OpsPilot 是一个基于 Go + Gin + React + TypeScript + Vite 的前后端分离自动化运维与任务调度平台。当前仓库处于 T0 基础工程阶段，目标是提供可直接运行、后续可扩展的开发骨架。

## 当前完成

- 后端：Go 1.26.2 + Gin，提供 `/health` 和 `/api/v1/ping`，统一响应结构为 `code/message/data/traceId`。
- 后端基础包：配置加载、结构化日志、traceId、CORS、统一错误/响应、GORM DB 封装、Redis 封装、JWT 管理器。
- 数据库：MySQL 8.0.39，初始 schema 覆盖认证、RBAC、Agent、主机、脚本、任务、日志、调度、监控、告警、Webhook、通知、审计。
- Agent：提供最小 `cmd/agent` 入口，后续 T2 接入注册、心跳、任务拉取和日志上传。
- 前端：React 19 + TypeScript + Vite，包含登录页、基础布局、仪表盘占位、请求封装、TanStack Query、Zustand。
- 部署：`deploy/docker-compose.yml` 提供 MySQL 8.0.39 和 Redis。

## 目录结构

```text
OpsPilot/
  server/
    cmd/
      api/
      agent/
    internal/
      app/
      config/
      platform/
      shared/
    migrations/
    docs/openapi/
  web/
    src/
      app/
      api/
      layouts/
      modules/
      shared/
      styles/
  deploy/
  docs/
```

## 环境要求

- Go 1.26.2
- Node.js 24.x
- npm 11.x
- MySQL 8.0.39
- Redis 7.x
- Docker Desktop 或兼容 Docker Compose 的环境

## 启动基础设施

```powershell
cd D:\+\1108026_rust_go\OpsPilot
docker compose -f .\deploy\docker-compose.yml up -d
```

MySQL 默认连接信息：

```text
host=localhost
port=3306
database=opspilot
user=opspilot
password=opspilot
root password=opspilot_root
charset=utf8mb4
collation=utf8mb4_0900_ai_ci
```

Redis 默认地址：

```text
localhost:6379
```

## 初始化数据库

当前迁移文件位于 `server/migrations`。如果本机安装了 MySQL 客户端，可执行：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000001_init_mysql_schema.up.sql
```

回滚初始 schema：

```powershell
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000001_init_mysql_schema.down.sql
```

## 启动后端

如果 `go version` 仍显示旧版本，请先把 Go 1.26.2 放到当前 PowerShell 的 PATH 最前面：

```powershell
$env:GOROOT=''
$env:Path = 'E:\install\GoWorkspace\pkg\go1.26.2\bin;' + $env:Path
$env:GOPROXY='https://goproxy.cn,direct'
go version
```

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
Copy-Item .\.env.example .\.env
go mod tidy
go run .\cmd\api
```

后端启动后访问：

```text
http://localhost:8080/health
```

如果本机 `8080` 已被占用，可以临时切换端口：

```powershell
$env:HTTP_ADDR=':18080'
go run .\cmd\api
```

运行测试：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
```

启动 Agent 骨架：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go run .\cmd\agent
```

## 启动前端

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
Copy-Item .\.env.example .\.env
npm install
npm run dev
```

前端默认地址：

```text
http://localhost:5173
```

构建检查：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm run build
```

## 后续开发顺序

1. T1：认证与 RBAC，接入真实注册、登录、JWT、刷新 Token、权限中间件和前端路由守卫。
2. T2：Agent 与主机管理，实现 Agent 注册、心跳、主机信息上报、在线离线扫描。
3. T3：脚本模板与任务执行，实现脚本版本、任务定义、Worker Pool、状态流转、超时、取消、重试。
4. T4：实时日志，实现 Agent 日志上传、后端日志入库、SSE 推送和前端日志终端。
5. T5-T8：调度、监控告警、Webhook、审计、OpenAPI、联调测试和部署完善。

## 开发约束

- 不直接把数据库 model 暴露给前端，后续业务模块必须使用 DTO。
- `handler` 只处理请求绑定、参数校验和响应；`service` 处理业务规则；`repository` 处理数据库访问。
- 所有 API 必须返回统一结构：`code`、`message`、`data`、`traceId`。
- 任务执行必须强制包含 `timeout_seconds`，并保存最终状态和执行记录。
- 涉及脚本执行、Webhook、Agent Token、JWT 的代码必须考虑安全、脱敏和审计。

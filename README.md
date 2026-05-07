# OpsPilot

> Language: 简体中文（当前） | [English](./README.en.md)

OpsPilot 是一个基于 Go + Gin + React + TypeScript 的自动化运维平台。当前代码已经完成从 Agent 管理原型到 T3/T4 执行链路的关键落地：脚本模板、任务执行、Agent 拉取执行、任务日志、SSE 实时日志、RBAC 与审计。

## 当前阶段

- T1 认证与 RBAC：登录、注册、JWT access/refresh token、用户/角色/权限、受保护路由。
- T2 Agent / Host：Agent 注册、Token 哈希存储、心跳、Agent/Host 列表、禁用、Token 吊销、离线扫描。
- T3 任务执行：脚本模板、任务创建、任务目标下发、Agent poll/claim、命令执行、结果上报、状态聚合。
- T4 实时日志：Agent stdout/stderr/system 日志上报、脱敏、分片存储、日志查询、SSE 实时消费。

## 已实现能力

- 统一 API 响应信封：`code`、`message`、`data`、`traceId`。
- 后端分层结构：handler 负责参数校验与响应；service 负责业务规则；repository 负责 SQL 访问。
- 脚本、任务、日志、Agent、Host 的 RBAC 权限控制。
- Agent Token 与用户 JWT 鉴权分离。
- 脚本版本管理：`script_versions.version_no`。
- 任务状态机：`pending -> queued -> running -> success|failed|timeout`，以及 `pending|queued -> canceled`。
- Agent 执行器：并发控制、超时、stdout/stderr 捕获、日志重试上传、结果上报。
- 前端页面：脚本管理、脚本审批队列、任务创建、任务列表、任务详情、实时日志、Agent/Host 管理、Enrollment Token 管理。
- 审计事件：登录、脚本创建/更新/禁用/审批、任务创建/取消、Agent 注册/禁用/Token 吊销、角色权限修改。

## 尚未实现

- 可视化工作流编排与 Cron 调度页面。
- “强制审批策略”闭环（例如：高风险脚本未审批不可执行）。
- Webhook 触发执行主链路。
- 告警与指标页面（当前以 schema 占位为主）。

## 环境要求

- Go 1.26.2
- Node.js 24.x
- npm 11.x
- MySQL 8.0.39
- Redis 7.x

请不要随意升级上述版本；仓库和 `docker-compose` 以这些版本为基线验证。

## 启动基础设施

```powershell
cd D:\+\1108026_rust_go\OpsPilot
docker compose -f .\deploy\docker-compose.yml up -d
```

默认 MySQL 配置：

```text
host=localhost
port=3306
database=opspilot
user=opspilot
password=opspilot
root password=opspilot_root
```

## 数据库迁移

按顺序执行：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000001_init_mysql_schema.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000002_seed_initial_auth_data.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000003_task_execution_security.up.sql
```

对应回滚脚本位于同目录下的 `.down.sql` 文件。

## 启动后端

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
Copy-Item .\.env.example .\.env
go mod tidy
go run .\cmd\api
```

关键环境变量：

```text
DATABASE_DSN=opspilot:opspilot@tcp(127.0.0.1:3306)/opspilot?charset=utf8mb4&parseTime=True&loc=Local
HTTP_ADDR=:8080
HTTP_ALLOW_ORIGIN=http://localhost:5173
AGENT_BOOTSTRAP_SECRET=dev-agent-bootstrap-secret
AGENT_REGISTRATION_ENABLED=true
AGENT_OFFLINE_SCAN_INTERVAL_SECONDS=45
```

## 启动前端

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
Copy-Item .\.env.example .\.env
npm install
npm run dev
```

默认前端地址：

```text
http://localhost:5173
```

## 启动 Agent

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
$env:AGENT_API_BASE_URL='http://localhost:8080'
$env:AGENT_BOOTSTRAP_SECRET='dev-agent-bootstrap-secret'
$env:AGENT_WORK_DIR='.tmp/agent-work'
$env:AGENT_POLL_INTERVAL_SECONDS='5'
$env:AGENT_HEARTBEAT_INTERVAL_SECONDS='30'
$env:AGENT_MAX_CONCURRENT_TASKS='2'
go run .\cmd\agent
```

当 `AGENT_TOKEN` 未设置时，Agent 会使用 `AGENT_BOOTSTRAP_SECRET` 注册，并把下发 token 写入 `AGENT_TOKEN_FILE`。数据库仅保存 token 哈希值，明文只在注册响应中返回一次。

使用一次性 Enrollment Token 注册：

```powershell
$env:AGENT_ENROLLMENT_TOKEN='<来自 UI 或 API 的 token>'
go run .\cmd\agent
```

## 任务执行演示

1. 使用初始化管理员登录：`admin / Admin@123456`。
2. 启动一个 Agent：`go run .\cmd\agent`。
3. 在 `Scripts` 新建脚本（例如：`echo hello && uname -a`）。
4. 在 `Tasks` 创建任务（可选脚本模板或 inline command）。
5. 选择在线 Agent 或 Host，并设置 `timeoutSeconds > 0`。
6. 打开任务详情页，观察 stdout/stderr/system 实时日志流。
7. 确认目标状态与任务最终状态正确收敛。

## 测试

后端：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
```

前端：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm run build
```

## 故障排查

- API 启动时报 `default workspace is not initialized`：
  请确认 `000002_seed_initial_auth_data.up.sql` 已执行到 `DATABASE_DSN` 指向的数据库。
- Agent 注册报 `401010 invalid agent bootstrap secret`：
  请确认 API 与 Agent 两侧 `AGENT_BOOTSTRAP_SECRET` 完全一致。
- Agent 注册报 `401013 invalid enrollment token`：
  token 可能已过期、被吊销，或达到 `max_uses` 次数上限；请重新签发。
- 前端日志流为空：
  请确认至少有一个目标处于 `running`，且当前用户具备 `task:log:read` 权限。
- 前端报 `Request failed`：
  请检查 `HTTP_ALLOW_ORIGIN` 与前端地址是否匹配，`VITE_API_BASE_URL` 是否指向正确后端。

## 安全说明

- 用户管理类 API 受 JWT + RBAC 保护。
- Agent 私有 API 仅接受 Agent Token，不接受仅用户 JWT 的访问。
- Agent token 仅哈希存储，支持吊销。
- 任务日志会对明显敏感字段做脱敏：`token`、`password`、`secret`、`authorization`、`bearer`。
- 生产环境下禁止 `HTTP_ALLOW_ORIGIN=*`，配置加载会拒绝该设置。
- 可通过 `AGENT_REGISTRATION_ENABLED=false` 关闭 bootstrap 注册入口。

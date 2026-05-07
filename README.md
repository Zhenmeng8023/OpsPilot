# OpsPilot

> 语言：简体中文（当前） | [English](./README.en.md)

OpsPilot 是一个基于 Go + Gin + React + TypeScript 的自动化运维平台。当前已经覆盖 Agent 接入、脚本模板、任务执行、实时日志、Cron 调度、Webhook 触发、基础指标采集、RBAC 与审计。

## 当前进展

- T1 认证与 RBAC：登录、注册、JWT access/refresh token、用户、角色、权限、受保护路由。
- T2 Agent / Host：Agent 注册、Token 哈希存储、心跳、Agent/Host 列表、禁用、Token 吊销、离线扫描。
- T3 任务执行：脚本模板、任务创建、目标下发、Agent poll/claim、命令执行、结果上报、状态聚合。
- T4 实时日志：Agent stdout/stderr/system 日志上报、脱敏、分片存储、查询 API、SSE 实时消费。
- T5 Cron 调度：schedules API、后台扫描器、前端 Schedules 页面，到期后复用现有任务执行链路。
- T6 Webhook：source/rule 管理、token 化触发入口、HMAC-SHA256 签名校验、防重放、基础限流、事件落库、生成 task_run。
- T7 Metrics / Alerts / Notifications：Agent 指标上报、host_metrics 持久化、前端 Metrics 页面、基础阈值告警规则、告警事件、站内通知、外部 webhook 类渠道发送与 delivery 记录。

## 已实现能力

- 统一 API 响应信封：`code`、`message`、`data`、`traceId`。
- 后端分层结构：handler 负责参数与响应，service 负责业务规则，repository/SQL 负责数据访问。
- 脚本、任务、日志、Agent、Host、Schedule、Webhook、Metrics 的 RBAC 权限控制。
- Agent Token 与用户 JWT 鉴权分离。
- 脚本版本管理与审批执行门禁：`approvalRequired=true` 时，仅最新审批状态为 `approved` 的版本可执行。
- 任务执行安全门禁：高危命令阻断、可配置 allowlist/denylist、timeout 最大值限制、Agent 独立工作目录、执行日志上传前脱敏、运行中取消时停止本地进程。
- tasks/scripts/logs 的分页、筛选与 total 返回。
- 前端权限矩阵：路由通过 `handle.meta.permission` 声明权限，统一守卫检查 permissions，侧边栏和关键按钮按权限显示。
- Cron 调度闭环：到期 schedule 复用 `task_runs` / `task_run_targets` / Agent poll / logs / result 链路。
- Webhook 触发闭环：`POST /api/v1/webhooks/trigger/:token` 需携带 `X-OpsPilot-Signature` 或 `X-Hub-Signature-256`，签名通过后匹配 rule 并生成 task_run。
- Metrics / Alerts 监控闭环：Agent 周期上报运行中任务数、逻辑 CPU、goroutine、运行时内存指标；后台按 metric threshold 规则生成或恢复告警。
- Notifications 通知闭环：支持 notification channel 管理、告警触发时生成 notifications 和 notification_deliveries、前端通知列表与标记已读，后台 dispatcher 会投递 webhook/dingtalk/wechat/slack 类型渠道。

## 尚未实现

- Cron 调度之外的可视化工作流编排。
- Webhook 的复杂 matcher 表达式、重放窗口细化和审计检索。
- Email/SMTP 通知、通知模板、通知重试管理界面；当前 webhook/dingtalk/wechat/slack 已支持按 channel URL 发送。

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

如需使用 Compose 启动 migrate/API/Web/Agent 全栈：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
Copy-Item .\deploy\.env.example .\deploy\.env -ErrorAction SilentlyContinue
docker compose --env-file .\deploy\.env -f .\deploy\docker-compose.yml --profile full up --build
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
TASK_COMMAND_ALLOW_PATTERNS=
TASK_COMMAND_DENY_PATTERNS=
SCHEDULE_SCAN_INTERVAL_SECONDS=30
ALERT_SCAN_INTERVAL_SECONDS=30
NOTIFICATION_DISPATCH_INTERVAL_SECONDS=15
NOTIFICATION_HTTP_TIMEOUT_SECONDS=10
```

生产环境安全约束：

- `APP_ENV=prod` 时，`/auth/register` 默认关闭，只有显式设置 `AUTH_PUBLIC_REGISTRATION_ENABLED=true` 才允许公开注册。
- `APP_ENV=prod` 时，`JWT_ACCESS_SECRET` 和 `JWT_REFRESH_SECRET` 不能为空、不能使用默认开发值、长度至少 32 字符。
- `HTTP_ALLOW_ORIGIN` 支持逗号分隔白名单；不匹配的 `Origin` 不会返回 CORS allow header。

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

当 `AGENT_TOKEN` 未设置时，Agent 会使用 `AGENT_BOOTSTRAP_SECRET` 或 `AGENT_ENROLLMENT_TOKEN` 注册，并把下发 token 写入 `AGENT_TOKEN_FILE`。

## 测试

后端：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
go vet ./...
```

前端：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm run build
```

迁移检查：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1 -SkipExecution
```

版本接口：

```text
GET /api/v1/version
```

完整手工验收流程见 [E2E Demo](./docs/e2e-demo.md)。

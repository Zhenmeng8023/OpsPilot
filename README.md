# OpsPilot

Release/security documents:
- [文档目录规范](./docs/README.md)
- [安全权限矩阵](./docs/security/permission-matrix.md) / [Security Permission Matrix](./docs/security/permission-matrix.en.md)
- [V0.7 安全发布检查清单](./docs/version/v0.7/security-release-checklist.md) / [V0.7 Security Release Checklist](./docs/version/v0.7/security-release-checklist.en.md)
- [V0.7 发布说明](./docs/version/v0.7/release-notes.md) / [V0.7 Release Notes](./docs/version/v0.7/release-notes.en.md)
- [V1.0 项目开发设计书](./docs/version/v1.0/product-development-design.md) / [V1.0 Product Development Design](./docs/version/v1.0/product-development-design.en.md)
- [V1.0 验收记录](./docs/version/v1.0/release-evidence.md) / [V1.0 Release Evidence](./docs/version/v1.0/release-evidence.en.md)
- [V1.0 部署手册](./docs/version/v1.0/deployment-guide.md) / [V1.0 Deployment Guide](./docs/version/v1.0/deployment-guide.en.md)
- [V1.0 验收手册](./docs/version/v1.0/acceptance-guide.md) / [V1.0 Acceptance Guide](./docs/version/v1.0/acceptance-guide.en.md)
- [V1.0 UI 规范](./docs/version/v1.0/ui-guidelines.md) / [V1.0 UI Guidelines](./docs/version/v1.0/ui-guidelines.en.md)
- [中长期软件开发设计规划](./docs/version/product-development-roadmap.md)
- [V1.1 下一版本设计](./docs/version/v1.1/product-development-design.md)
- [V1.1 运维加固演练手册](./docs/version/v1.1/operations-hardening.md)

> 语言：简体中文（当前） | [English](./README.en.md)

OpsPilot 是一个基于 Go + Gin + React + TypeScript 的自动化运维平台。当前已经覆盖 Agent 接入、脚本模板、任务执行、实时日志、Cron 调度、Webhook 触发、指标采集、Alerts、Notifications、RBAC 与审计。

## 当前进展

- T1 认证与 RBAC：登录、注册、JWT access/refresh token、用户、角色、权限、受保护路由。
- T2 Agent / Host：Agent 注册、Token 哈希存储、心跳、Agent/Host 列表、禁用、Token 吊销、离线扫描。
- T3 任务执行：脚本模板、任务创建、目标下发、Agent poll/claim、命令执行、结果上报、状态聚合。
- T4 实时日志：Agent stdout/stderr/system 日志上报、脱敏、分片存储、查询 API、SSE 实时消费。
- T5 Cron 调度：schedules API、preview、misfire policy、trigger history、后台扫描器、前端 Schedules 页面，到期后复用现有任务执行链路。
- T6 Webhook：source/rule 管理、token 化触发入口、独立 signing secret、HMAC-SHA256 签名校验、防重放、基础限流、事件落库、matcher 调试、生成 task_run。
- T7 Metrics / Alerts / Notifications：Agent OS 指标上报、host_metrics 持久化、前端 Metrics 页面、阈值告警规则、告警事件、站内/Email/webhook 类通知、delivery 查询、重试和 test send。

## 已实现能力

- 统一 API 响应信封：`code`、`message`、`data`、`traceId`。
- 后端分层结构：handler 负责参数与响应，service 负责业务规则，repository/SQL 负责数据访问。
- 脚本、任务、日志、Agent、Host、Schedule、Webhook、Metrics 的 RBAC 权限控制。
- Agent Token 与用户 JWT 鉴权分离。
- 脚本版本管理与审批执行门禁：`approvalRequired=true` 时，仅最新审批状态为 `approved` 的版本可执行。
- 任务执行安全门禁：高危命令阻断、可配置 allowlist/denylist、timeout 最大值限制、Agent 独立工作目录、执行日志上传前脱敏、运行中取消时停止本地进程。
- tasks/scripts/logs 的分页、筛选与 total 返回。
- 前端权限矩阵：路由通过 `handle.meta.permission` 声明权限，统一守卫检查 permissions，侧边栏和关键按钮按权限显示。
- Cron 调度闭环：支持 schedule preview、misfire policy、trigger history，并复用 `task_runs` / `task_run_targets` / Agent poll / logs / result 链路。
- Webhook 触发闭环：`POST /api/v1/webhooks/trigger/:token` 需携带 `X-OpsPilot-Signature` 或 `X-Hub-Signature-256`，签名通过后匹配 rule 并生成 task_run。
- Metrics / Alerts 监控闭环：Agent 上报真实 OS 指标和运行时指标；后台按 metric threshold 规则生成、确认、静默、恢复告警。
- Notifications 通知闭环：支持 notification channel 管理、SMTP Email、webhook 类外部投递、delivery 查询/重试/test send、前端通知列表与标记已读。

## V1.0 当前状态

- 当前仓库已经满足 Release Evidence 和验收手册中记录的 V1.0 验收门槛，根版本号已收口为 `1.0.0`。
- GitHub Actions 已在 `master` 与 `v1.0.0-beta.1` tag 上完成远端绿色验证。
- Webhook matcher 已补齐数组通配、`payload_exists`、`payload_not_equals`、`payload_regex` 等完整能力。

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

推荐一键脚本（包含 schema_migrations 记录与校验）：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migrations.ps1 -Mode apply -HostName 127.0.0.1 -Port 3306 -Database opspilot -User opspilot -Password opspilot
.\scripts\migrations.ps1 -Mode verify -HostName 127.0.0.1 -Port 3306 -Database opspilot -User opspilot -Password opspilot
```

如需手工按顺序执行：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000001_init_mysql_schema.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000002_seed_initial_auth_data.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000003_task_execution_security.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000004_v07_webhook_security.up.sql
mysql -h 127.0.0.1 -P 3306 -uopspilot -popspilot opspilot < .\server\migrations\000005_v10_productization_bundle.up.sql
```

新环境默认执行根目录的 `000001` 到 `000005_v10_productization_bundle`。原始 V1.0 细粒度增量迁移已归档到 `server/migrations/archive/v1.0-incremental/`，用于历史追溯和旧链路核对。

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
METRIC_ROLLUP_INTERVAL_SECONDS=300
METRIC_RETENTION_INTERVAL_SECONDS=86400
METRIC_DETAIL_RETENTION_DAYS=7
METRIC_ROLLUP_RETENTION_DAYS=90
NOTIFICATION_DISPATCH_INTERVAL_SECONDS=15
NOTIFICATION_HTTP_TIMEOUT_SECONDS=10
```

如果数据库服务时区与 API 进程本地时区不同，不要继续使用 `loc=Local`。请将 DSN 中的 `loc` 改成与数据库一致的 IANA 时区，例如 `loc=Asia%2FShanghai`，否则 wait/schedule 等依赖时间比较的功能在 Docker 或跨时区环境下可能出现偏差。

生产环境安全约束：

- `APP_ENV=prod` 时，`/auth/register` 默认关闭，只有显式设置 `AUTH_PUBLIC_REGISTRATION_ENABLED=true` 才允许公开注册。
- `APP_ENV=prod` 时，`JWT_ACCESS_SECRET` 和 `JWT_REFRESH_SECRET` 不能为空、不能使用默认开发值、长度至少 32 字符。
- `APP_ENV=prod` 时，`AGENT_BOOTSTRAP_SECRET`、`BOOTSTRAP_ADMIN_PASSWORD`、`SECRET_ENCRYPTION_KEY` 同样不能为空、不能使用默认开发值、长度至少 32 字符。
- `HTTP_ALLOW_ORIGIN` 支持逗号分隔白名单；不匹配的 `Origin` 不会返回 CORS allow header。
- `APP_ENV=prod` 时，`HTTP_ALLOW_ORIGIN` 不能包含 `localhost`、`127.0.0.1`、`::1`。
- `APP_ENV=prod` 时，`DATABASE_DSN` 必须包含 `parseTime=True`。

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
.\scripts\openapi-router-check.ps1
.\scripts\govulncheck.ps1
```

npm 漏洞检查（High/Critical 门禁）：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
npm audit --audit-level=high --omit=dev --registry=https://registry.npmjs.org
```

版本接口：

```text
GET /api/v1/version
```

完整手工验收流程见 [E2E Demo](./docs/overview/e2e-demo.md)。

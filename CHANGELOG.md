# Changelog

> Language: 简体中文（当前） | [English](./CHANGELOG.en.md)

## Unreleased

### V0.7 发布骨架

- 新增 GitHub Actions CI：后端测试、`go vet`、迁移配对检查、前端构建和 Docker build。
- 新增根目录 `VERSION`、构建版本注入和公开接口 `GET /api/v1/version`。
- 新增 API/Web Dockerfile，并扩展 Compose `full` profile 支持 migrate/API/Web/Agent 全栈启动。
- 新增迁移检查脚本 `scripts/migration-check.ps1`。
- OpenAPI 同步到当前 T5-T8 核心路由，并新增 `docs/e2e-demo.md` 手工验收流程。

### V0.7 安全基线

- 新增 `AUTH_PUBLIC_REGISTRATION_ENABLED`，生产环境默认禁止公开注册。
- 新增生产 JWT secret 校验：拒绝空值、默认开发值和过短密钥。
- CORS 改为逗号分隔白名单匹配，不匹配的 `Origin` 不返回 allow header。

### T5 Cron 调度

- 新增 schedules 后端模块：支持 Cron 周期任务创建、分页列表、暂停、恢复、禁用。
- 新增后台调度扫描器：按 `SCHEDULE_SCAN_INTERVAL_SECONDS` 扫描到期 schedule，并复用现有 `task_runs` / `task_run_targets` / Agent poll / logs / result 执行链路。
- 新增前端 Schedules 页面：可选择现有任务定义创建 Cron 调度，并查看 next/last fire 时间。
- 新增前端路由权限矩阵：路由使用 `handle.meta.permission` 声明权限，守卫统一检查登录用户 permissions，侧边栏与操作按钮按权限显示。

### T6 Webhook 触发

- 新增 Webhook source/rule 管理 API 与前端 Webhooks 页面。
- 新增公开触发入口 `POST /api/v1/webhooks/trigger/:token`，通过 source token 识别来源。
- 触发时写入 `webhook_events` 与 `webhook_event_matches`，匹配 rule 后复用现有任务执行链路生成 `task_runs`。
- 增加基础防护：`X-OpsPilot-Signature` / `X-Hub-Signature-256` HMAC-SHA256 签名校验、`X-Delivery-Id` 防重放、每 source 每分钟限流、payload hash 与请求头落库。

### T7 Metrics 监控

- 新增 Agent metrics 上报接口 `POST /api/v1/agent/metrics`，写入 `host_metrics`。
- Agent 心跳周期同步上报运行中任务数、逻辑 CPU、goroutine、运行时内存指标。
- 新增管理端 `GET /api/v1/metrics/hosts` 与前端 Metrics 页面，用于查看最近主机/Agent 指标。
- 新增 alert rules / alerts 管理 API 与后台扫描器，支持 metric threshold 规则生成 firing 告警并在指标恢复时自动 resolve。
- Metrics 页面新增告警规则创建、当前 firing 告警和规则列表。

### T8 Notifications 通知

- 新增 notification channel / notification 管理 API 与前端 Notifications 页面。
- 告警首次 firing 时自动生成 `notifications` 和 `notification_deliveries`。
- 站内通知 channel 会立即标记 delivery success；webhook/dingtalk/wechat/slack 渠道由后台 dispatcher 读取 pending delivery 并按 URL 投递。
- 前端支持通知列表、未读筛选、标记已读、渠道列表和 webhook 类渠道 URL 配置。

### T1 认证与 RBAC

- 新增登录、注册、刷新/登出、当前用户、用户管理、角色管理、权限管理 API。
- 新增 JWT 中间件和管理端权限校验。
- 初始化内置 `admin/member` 角色及基础权限集合。

### T2 Agent 与 Host

- 新增 Agent 注册、token 哈希存储、心跳、Agent 列表、Host 列表、Agent 禁用、token 吊销、离线扫描。
- 新增根据 Agent 状态聚合 Host 状态。
- 在 API 进程中新增后台离线扫描任务。

### T3 脚本与任务执行

- 新增脚本模板 CRUD、禁用、`script_versions` 版本追踪。
- 新增基于脚本模板或 inline command 的任务创建。
- 新增任务 run/target 创建、状态机、取消、Agent poll/claim、结果上报、聚合状态刷新。
- 将 `cmd/agent` 升级为执行器：并发控制、超时处理、stdout/stderr 捕获、结果回传。

### T4 实时日志

- 新增 Agent stdout/stderr/system 日志上报。
- 新增任务日志分片持久化（带顺序号与源时间戳）。
- 新增管理端日志查询 API 与 SSE 流式接口。
- 新增前端任务日志终端（暂停、复制、清空、目标过滤）。

### 安全与审计

- 新增 `token/password/secret/authorization/bearer` 模式脱敏。
- 新增登录、脚本、任务、Agent、角色权限变更的审计记录。
- 新增生产环境 CORS 防护与 Agent 注册开关。

### 工作流与界面

- 新增 Agent Enrollment Token 管理界面：创建、一次性展示、复制、列表、撤销。
- 新增脚本审批界面：待审批队列、approve/reject 操作、脚本编辑页发起审批。
- OpenAPI 同步补齐 Enrollment Token 与脚本审批相关接口。
- 迁移种子同步补齐 `script:approve` 权限，避免仅跑迁移时审批权限缺失。
- 新增脚本审批执行门禁：任务创建时，`approvalRequired=true` 且最新审批状态非 `approved` 的脚本将被后端拒绝执行。
- 新增审批状态可视化：脚本列表增加审批状态列，任务创建页脚本下拉显示审批状态并对不可执行脚本给出阻断提示。
- 新增任务执行安全门禁：阻断 `rm/del/format/shutdown/reboot/mkfs/Remove-Item` 等高危命令，限制 timeout 最大值为 3600 秒。
- 命令策略支持环境变量配置：`TASK_COMMAND_ALLOW_PATTERNS` 与 `TASK_COMMAND_DENY_PATTERNS` 可按正则追加 allowlist / denylist。
- Agent 执行器改为每个任务使用独立工作目录，并在上传 stdout/stderr/system 日志前进行脱敏。
- 任务取消补齐运行中停止链路：后端新增 `canceling` 状态与 Agent 目标状态查询接口，Agent 发现取消请求后会停止本地命令并上报 `canceled`。

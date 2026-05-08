# OpsPilot MySQL 8.0.39 数据库设计

本文档说明 OpsPilot 面向长期演进的 MySQL 数据库设计。当前 schema 对应迁移文件：

- `server/migrations/000001_init_mysql_schema.up.sql`
- `server/migrations/000001_init_mysql_schema.down.sql`

## 设计原则

- 内部主键使用 `BIGINT UNSIGNED AUTO_INCREMENT`，便于 join、索引和 MySQL 运维。
- 外部 API 标识使用 `uid CHAR(26)`，预期存 ULID，避免把递增主键暴露给前端和 Agent。
- 默认字符集为 `utf8mb4`，排序规则为 `utf8mb4_0900_ai_ci`。
- 所有核心业务表保留 `created_at`、`updated_at`、必要时保留 `deleted_at`，便于软删除和审计。
- 按 `workspace_id` 预留团队/空间边界，当前可以只初始化一个默认 workspace，后续可自然扩展多团队。
- 任务定义、脚本版本、任务运行、目标、尝试、日志分层保存，保证历史运行可回放、可审计。
- Redis 只做短期状态、锁、限流和广播，不作为任务最终状态的唯一来源。

## 表分组

| 分组 | 表 |
| --- | --- |
| 空间与用户 | `workspaces`, `users`, `workspace_members` |
| 权限 | `permissions`, `roles`, `role_permissions`, `user_roles`, `refresh_tokens`, `login_logs` |
| 通用配置 | `system_settings`, `tags`, `resource_tags`, `idempotency_keys`, `outbox_events` |
| Agent 与主机 | `host_groups`, `hosts`, `host_group_members`, `agents`, `agent_tokens`, `agent_heartbeats`, `agent_commands` |
| 脚本 | `script_templates`, `script_versions`, `script_approvals` |
| 任务 | `tasks`, `task_targets`, `task_runs`, `task_run_targets`, `task_run_attempts`, `task_run_logs`, `task_run_events`, `task_run_artifacts` |
| 调度 | `schedules`, `schedule_triggers` |
| 监控 | `metric_definitions`, `host_metrics`, `service_checks`, `service_check_results` |
| 告警与通知 | `alert_rules`, `alerts`, `alert_events`, `notification_channels`, `notifications`, `notification_deliveries` |
| Webhook | `webhook_sources`, `webhook_rules`, `webhook_events`, `webhook_event_matches` |
| 审计 | `audit_logs` |

## 核心建模说明

### 认证与 RBAC

`users` 是全局用户表，`workspace_members` 表示用户在哪个 workspace 内有效。`roles` 支持 `workspace_id` 为空的系统角色模板，也支持 workspace 内自定义角色。`roles` 使用 `IFNULL(workspace_id, 0)` 函数唯一索引，保证系统角色模板和 workspace 内角色都能按 `code` 正确唯一。权限编码使用 `permission.code`，例如：

- `agent.read`
- `agent.disable`
- `script.write`
- `task.run`
- `task.cancel`
- `webhook.manage`
- `audit.read`

`refresh_tokens` 只保存 token hash，不保存明文 token。后续实现 refresh token 轮换时，使用 `token_family` 识别同一登录会话链路。

### Agent 与主机

`hosts` 表示被管理主机，`agents` 表示部署在主机上的执行端。二者分开是为了支持：

- 一个 host 重新安装 Agent 后保留主机历史。
- 一个 host 临时无 Agent 但仍保留资产信息。
- 后续支持多 Agent 或执行器类型时不破坏主机模型。

`agent_tokens` 支持 token 轮换和吊销。`agent_heartbeats` 保存心跳历史，但它是高增长表，建议后续只保留短周期原始数据，并做归档或聚合。

### 脚本版本

`script_templates` 保存脚本模板的业务身份，`script_versions` 保存不可变脚本内容。任务引用 `script_version_id`，这样即使模板后来被修改，历史任务仍能追溯当时执行的真实脚本内容。

高风险脚本可以通过 `approval_required` 和 `script_approvals` 扩展审核流程。

### 任务执行

任务执行分为五层：

- `tasks`：任务定义。
- `task_targets`：任务默认目标。
- `task_runs`：一次触发产生的运行批次。
- `task_run_targets`：一次运行内的具体目标。
- `task_run_attempts`：某个目标的一次执行尝试，重试会产生新的 attempt。

日志写入 `task_run_logs`，使用 `(attempt_id, sequence)` 保证 Agent 重传日志时可幂等去重。状态变化写入 `task_run_events`，附件或产物写入 `task_run_artifacts`。

### 调度

`schedules` 支持 `once`、`interval`、`cron`。`schedule_triggers` 记录每次计划触发与实际触发，便于排查错过调度、重复调度和调度延迟。

### 监控告警

`host_metrics` 是时序型高增长表，初期可直接写 MySQL；当数据量明显增长后，建议：

- 保留近 7-30 天原始数据。
- 生成分钟级/小时级聚合表。
- 对 `host_metrics` 按月或按周分区。
- 如果指标量继续上升，再迁移到 Prometheus、VictoriaMetrics 或 ClickHouse。

告警由 `alert_rules` 生成 `alerts`，状态变化进入 `alert_events`。`alerts.open_fingerprint` 是生成列，只对 `firing`、`acknowledged`、`silenced` 等未关闭状态生效，保证同一 workspace 下同一 fingerprint 同时只有一条未关闭告警，同时允许保留多条已恢复历史。通知发送状态由 `notification_deliveries` 跟踪，便于失败重试。

### Webhook

`webhook_sources` 保存来源和密钥 hash，`webhook_rules` 负责匹配事件并触发任务，`webhook_events` 保存请求记录。失败签名、重放请求、未匹配事件也应入库，便于安全审计。

### 审计

`audit_logs` 记录关键行为：

- 登录和认证失败。
- 角色权限变更。
- Agent 注册、禁用、token 轮换。
- 脚本模板和脚本版本变更。
- 任务执行、取消、重试。
- Webhook 触发和签名失败。
- 告警确认、恢复、静默。

审计表是长期保留表，建议按月分区或定期归档到冷存储。

## 高增长表和后续治理

| 表 | 增长原因 | 初期策略 | 长期策略 |
| --- | --- | --- | --- |
| `task_run_logs` | 每次任务实时输出日志 | MySQL 入库，按 run/attempt/sequence 查询 | 按月分区、压缩、对象存储归档 |
| `host_metrics` | Agent 周期上报指标 | MySQL 保存近周期原始数据 | 聚合表、分区、迁移到时序库 |
| `agent_heartbeats` | Agent 高频心跳 | 保留近期心跳历史 | 只保留最近 N 天，旧数据归档 |
| `audit_logs` | 所有关键操作审计 | MySQL 长期保存 | 按月分区，冷归档 |
| `webhook_events` | 外部事件触发频繁 | 保存 payload 摘要和必要载荷 | 大 payload 转对象存储 |

## 索引策略

核心查询路径已经建立组合索引：

- Agent 在线状态：`agents(workspace_id, status, last_heartbeat_at)`
- 主机状态：`hosts(workspace_id, status)`
- 任务运行列表：`task_runs(workspace_id, status, created_at)`
- 任务运行日志：`task_run_logs(run_id, sequence)` 和 `task_run_logs(attempt_id, sequence)`
- 调度扫描：`schedules(workspace_id, status, next_fire_at)`
- 指标趋势：`host_metrics(host_id, metric_code, collected_at)`
- 告警列表：`alerts(workspace_id, status, severity)`
- 审计检索：`audit_logs(workspace_id, created_at)` 和 `audit_logs(workspace_id, resource_type, resource_id, created_at)`

后续不要盲目加单列索引，应根据真实慢查询和页面查询条件追加组合索引。

## MySQL 8.0.39 约束说明

- 使用 `CHECK` 约束限制核心状态枚举，MySQL 8.0.16+ 会执行 `CHECK`。
- 使用 `JSON` 保存变量 schema、任务参数、事件 payload、匹配规则等半结构化数据。
- 不在 JSON 字段上默认建立函数索引；当某个 JSON 路径成为高频过滤条件时，再增加 generated column + index。
- MySQL 唯一索引允许多条 `NULL`，所以 `roles`、`system_settings` 这类支持全局作用域的表使用 `IFNULL(workspace_id, 0)` 函数唯一索引来表达唯一约束。
- MySQL 不支持原生 partial unique index，所以 `alerts` 使用 `open_fingerprint` 生成列模拟“仅未关闭告警唯一”的约束。
- 外键用于保护核心一致性；日志和事件类表多数使用级联删除，审计类表使用 `SET NULL` 保留历史。

## 设计复核结论

当前 schema 足以支撑 README 中规划的 T1-T8 主线功能：认证与 RBAC、Agent 与主机管理、脚本版本、任务执行、实时日志、调度、监控告警、Webhook、通知和审计。后续真正进入高并发或长周期生产使用前，仍需要按真实数据规模补充分区、归档、聚合表、seed 数据和正式迁移工具；这些属于演进治理，不是当前业务模型缺口。

## 后续迁移建议

1. T1 实现认证时，补充默认 workspace、默认管理员、基础权限和内置角色的 seed 数据。
2. T2 实现 Agent 时，补充 Agent token 轮换和心跳清理任务。
3. T3 实现任务时，严格围绕 `task_runs -> task_run_targets -> task_run_attempts -> task_run_logs` 写入，不要只更新 `tasks.status`。
4. T4 实时日志上线后，压测 `task_run_logs` 写入速度，再决定是否增加批量插入和日志保留策略。
5. T5 监控上线前，明确指标上报频率，避免 `host_metrics` 过早膨胀。
6. T8 联调阶段引入正式迁移工具，例如 `golang-migrate` 或 `goose`。

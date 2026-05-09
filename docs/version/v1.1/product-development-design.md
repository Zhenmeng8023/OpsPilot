# OpsPilot V1.1 开发规划说明书

生成日期：2026-05-09

关联文档：

- [中长期软件开发设计规划](../product-development-roadmap.md)
- [V1.0 项目开发设计书](../v1.0/product-development-design.md)
- [V1.0 发布验收证据](../v1.0/release-evidence.md)
- [V1.0 部署手册](../v1.0/deployment-guide.md)
- [V1.0 UI 规范](../v1.0/ui-guidelines.md)
- [跨版本 UML 总览](../../architecture/uml/core-uml.md)
- [安全权限矩阵](../../security/permission-matrix.md)

## 1. 文档目的

本文是 OpsPilot V1.1 的完整开发规划说明书，用于把中长期路线图中的 V1.1 方向落到可执行的工程计划。它不只是功能列表，还定义以下内容：

- V1.1 的产品定位、版本边界和不做事项。
- P0/P1/P2 的拆分、依赖、验收门槛和延期规则。
- 后端、前端、数据模型、OpenAPI、测试、发布证据的开发要求。
- 里程碑、风险控制、回退策略和硬停止条件。

V1.1 的核心目标是把 V1.0 已经闭环的自动化运维控制平面推向真实生产值班可用，而不是开启新的主业务。

## 2. 当前基线

### 2.1 V1.0 已完成能力

V1.0 已经形成 OpsPilot 的第一条完整主业务链路：

```text
人工 / Cron / Webhook 触发
  -> Workflow 编排
  -> Agent / Host 执行
  -> 日志 / 指标 / 告警 / 通知
  -> RBAC / Audit / Release Evidence
```

主要能力包括：

- Agent 注册、心跳、主机标签、任务执行、日志流。
- Script Template、Task、Task Target、Run 日志。
- Workflow definition、workflow run、node run、版本化执行。
- Cron、Webhook、Manual 三类触发。
- Metrics、Alerts、Notifications、Incident 投影。
- RBAC、Audit、Secret、权限矩阵。
- React 控制台、Go API、MySQL/Redis、Docker Compose 部署。

### 2.2 V1.0 暴露的生产化缺口

V1.0 能证明主链路可跑通，但生产值班还缺少以下能力：

| 缺口 | 生产影响 | V1.1 应对 |
| --- | --- | --- |
| 缺少跨模块 trace | 触发失败、执行失败、通知失败之间难以串联 | 建设 Trace Center |
| Workflow 失败恢复不够可解释 | retry/cancel 影响范围不清晰，容易重复执行 | 强化 retry plan、cancel report、action history |
| UI 验收依赖人工和 build | 页面能否打开、窄屏是否溢出、关键页面是否空白缺少证据 | 引入 Playwright UI smoke |
| Agent Fleet 规模视角不足 | Agent 多起来后无法快速判断版本、异常原因、影响范围 | P1 加固 Agent Fleet 视图 |
| 告警治理仍偏基础 | routing/suppression 命中原因和噪声趋势不够透明 | P1 加固告警与 Incident 治理 |
| 安全审计颗粒度还可提升 | secret rotation、权限变更差异、audit export 完整性不足 | P1 加固安全与合规 |

## 3. 版本定位

### 3.1 一句话定位

V1.1 是“生产化加固版本”：围绕 Trace、Workflow 可靠性和 UI 自动验收，让 OpsPilot 从“功能闭环”升级为“值班可用、失败可查、恢复可控、发布可验证”。

### 3.2 核心目标

V1.1 必须达成以下目标：

1. 值班人员可以从一个 trace 还原一次触发、执行、通知、审计链路。
2. Workflow 失败后可以安全重试、取消、复盘，并能解释每次动作的影响范围。
3. 关键 UI 页面在 desktop 与 narrow viewport 下有自动 smoke 和截图证据。
4. 新增能力不破坏 V1.0 数据、接口和部署方式。
5. P1 能力只能作为加固项进入，不能挤压 P0 发布。

### 3.3 不做事项

V1.1 明确不做：

- 不引入 Service Catalog、Request Center、运维服务交付等 V2.0 新主业务。
- 不做多租户、workspace/team 强隔离、复杂计费。
- 不做 AI 自动修复作为核心执行能力。
- 不做微服务拆分、消息总线重构或外部时序数据库替换。
- 不做 Agent 自动升级生产能力，只允许留下协议草案或 P2 预研。
- 不把 P1 的 Agent Fleet、Incident、安全合规作为无条件发布门槛。

## 4. 设计原则

1. 先可诊断，再扩复杂能力。任何新增执行链路都必须能被 trace 和 audit 观测。
2. 服务端状态机优先。retry、cancel、approve、suppress 等动作必须以服务端状态为准。
3. 默认兼容 V1.0。新增字段应允许为空或提供默认值；迁移必须有 up/down。
4. UI 证据自动化。V1.1 起关键页面需要 smoke、截图和失败 artifact。
5. P0 可独立发布。P1/P2 未完成不得阻塞 V1.1。
6. 权限、secret、审计不后补。涉及敏感数据和执行动作的能力必须同步设计权限与审计。
7. 性能目标写入验收。Trace 查询、Workflow 列表、UI smoke 都必须有可验证门槛。
8. 每个新增表声明 retention 或增长策略，避免事件类数据无限膨胀。

## 5. 范围分层

### 5.1 P0 发布承诺

P0 是 V1.1 的发布承诺，必须全部完成：

| Epic | 名称 | 交付目标 |
| --- | --- | --- |
| OP-11-TRACE | Trace Center | 跨 workflow、task、webhook、schedule、notification、audit、alert/incident 的排障入口 |
| OP-11-WF-RELIABILITY | Workflow 运行可靠性 | retry plan、cancel propagation、action history、definition diff、node output summary |
| OP-11-UI-SMOKE | Playwright UI Smoke | 关键页面 desktop/narrow 自动打开、截图、失败 artifact |

### 5.2 P1 加固候选

P1 只能在 P0 已闭环后进入，不得阻塞 V1.1：

| Epic | 名称 | 交付目标 |
| --- | --- | --- |
| OP-11-AGENT-FLEET | Agent Fleet 规模化视图 | version distribution、stale reason、diagnostics diff、维护窗口 preview |
| OP-11-INCIDENT-GOV | 告警与 Incident 治理 | routing explanation、noise report、incident 手动合并/关闭 |
| OP-11-SECURITY-HARDENING | 安全与合规加固 | secret rotation history、permission diff、audit export hash |

### 5.3 P2 探索项

P2 只允许输出设计草案、接口边界或 mock，不进入 V1.1 发布承诺：

- Workflow template marketplace 最小信息架构。
- 外部时序数据库 adapter 草案。
- OIDC/SSO 登录预研。
- Agent 自动升级协议草案。
- ChatOps 告警确认入口草案。

## 6. 用户与场景

### 6.1 目标用户

| 用户 | 关心的问题 | V1.1 提供的能力 |
| --- | --- | --- |
| 值班工程师 | 任务为什么失败、通知有没有发出、谁触发了操作 | Trace Center、Workflow action history、notification timeline |
| 运维负责人 | 哪些 workflow 不可靠、哪些 Agent 异常、告警是否噪声过高 | Workflow search、Agent Fleet、Noise report |
| 安全/审计人员 | 高危操作是否可追踪、secret 是否轮换、权限变更影响是什么 | Audit 增强、secret rotation history、permission diff |
| 前端/发布负责人 | 关键页面是否能打开、窄屏是否溢出、发布证据是否完整 | Playwright UI smoke、截图 artifact、release evidence |

### 6.2 核心用户旅程

#### 6.2.1 触发失败排障

```text
用户收到告警或发现 workflow run failed
  -> 从 run detail 复制 traceId
  -> 打开 Trace Center
  -> 查看 webhook/schedule/manual trigger
  -> 查看 workflow node 和 task target 状态
  -> 查看 notification delivery 与 audit action
  -> 判断失败原因并重试或取消
```

#### 6.2.2 Workflow 安全重试

```text
workflow run failed
  -> 查看 failed node 与 upstream/downstream
  -> 打开 retry plan preview
  -> 确认哪些节点会跳过、哪些节点会重跑
  -> 提交 retry action
  -> action history 和 audit 写入记录
  -> Trace Center 更新 retry timeline
```

#### 6.2.3 发布 UI 回归

```text
CI 或本地启动前后端
  -> API seed 创建最小数据
  -> Playwright 打开关键页面
  -> desktop/narrow 截图
  -> 检查 runtime error、空白页、明显文本溢出、基础 a11y
  -> 失败时保留 screenshot/trace
  -> release evidence 引用 artifact
```

## 7. OP-11-TRACE：Trace Center

### 7.1 目标

建设全局 Trace Center，使用户可以通过 `traceId`、workflow run id、task run id、task target id、webhook event id、schedule trigger id、alert id、incident id、notification id 查到一次操作链路。

### 7.2 Trace 对象定义

V1.1 的 trace 不是替代所有业务表的新模型，而是一个跨模块查询视角。核心字段建议：

| 字段 | 说明 |
| --- | --- |
| `trace_id` | 链路唯一标识；优先贯穿 trigger、workflow、task、notification、audit |
| `source_type` | `manual`、`schedule`、`webhook`、`workflow`、`task`、`alert`、`incident` |
| `source_id` | 源对象 ID |
| `event_type` | 标准事件类型，例如 `workflow.started`、`task.failed` |
| `severity` | `info`、`warning`、`error` |
| `occurred_at` | 事件发生时间 |
| `actor_type` | `user`、`system`、`agent`、`webhook`、`scheduler` |
| `actor_id` | 操作者或系统来源 |
| `entity_type` | 关联业务实体类型 |
| `entity_id` | 关联业务实体 ID |
| `summary` | 时间线摘要 |
| `metadata` | 脱敏后的结构化信息 |

### 7.3 数据来源

Trace Center 首版应从现有表按需聚合，不强制所有事件先写入 `trace_events`。数据来源包括：

- Workflow run、workflow node run、workflow run action。
- Task run、task target、task log 摘要。
- Webhook event、schedule trigger。
- Notification delivery、delivery retry。
- Audit log。
- Alert group、incident timeline。
- Agent 状态和执行回执。

### 7.4 `trace_events` 策略

`trace_events` 是可选缓存表，不是 V1.1 的强制唯一事实源。

使用规则：

- 如果按需聚合 p95 小于 1 秒，可以先不启用缓存表。
- 如果常规数据量下 p95 超过 1 秒，必须引入 `trace_events` 或分页 timeline。
- `trace_events` 必须能从业务表重建，不能成为不可恢复的唯一事实。
- `trace_events` 必须声明 retention 配置，默认只保留配置项，不硬编码删除周期。

建议表：

```sql
trace_events(
  id,
  trace_id,
  source_type,
  source_id,
  event_type,
  severity,
  actor_type,
  actor_id,
  entity_type,
  entity_id,
  summary,
  metadata_json,
  occurred_at,
  created_at
)
```

### 7.5 后端接口

P0 必须提供：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/traces/:id` | 根据 trace id 或业务 id 返回链路摘要和 timeline |
| `GET` | `/api/v1/traces/:id/events` | 分页返回 timeline 事件 |
| `GET` | `/api/v1/traces/lookup` | 根据 entity type/id 查 trace |

响应结构建议：

```json
{
  "traceId": "trc_xxx",
  "status": "failed",
  "startedAt": "2026-05-09T10:00:00Z",
  "endedAt": "2026-05-09T10:03:00Z",
  "source": {
    "type": "webhook",
    "id": "evt_123"
  },
  "entities": [
    { "type": "workflow_run", "id": "run_123", "status": "failed" }
  ],
  "timeline": [
    {
      "time": "2026-05-09T10:00:01Z",
      "type": "workflow.started",
      "severity": "info",
      "summary": "Workflow started by webhook",
      "entityType": "workflow_run",
      "entityId": "run_123"
    }
  ]
}
```

### 7.6 前端页面

新增 `Trace Center` 页面：

- 顶部搜索框支持 traceId 和业务对象 ID。
- 左侧 summary：状态、来源、耗时、触发人、关联对象。
- 主体 timeline：按时间排序展示 trigger、workflow、task、notification、audit。
- 右侧 inspector：展示选中事件的脱敏 metadata、跳转链接、相关日志摘要。
- 支持从 Workflow Run、Task Run、Webhook Event、Alert、Incident、Notification Delivery、Audit 页面一键跳转。

UI 要求：

- timeline 事件高度稳定，不因摘要过长撑破布局。
- metadata 默认折叠，敏感字段默认脱敏。
- 复制 traceId 使用 icon button，提供 tooltip。
- 窄屏下 summary、timeline、inspector 改为纵向布局。

### 7.7 权限与脱敏

Trace Center 不能因为聚合视图扩大权限。

规则：

- 用户只能看到自己已有权限可访问的实体。
- 没有权限的事件可以显示“存在不可见事件”，但不能泄露 payload。
- webhook header、secret value、notification token、task output 中的敏感字段必须脱敏。
- 查看 trace、导出 trace、访问敏感 metadata 都要写 audit。

### 7.8 验收标准

P0 验收：

- manual workflow、schedule workflow、webhook workflow 都能生成可查询 trace。
- 失败节点、取消 run、notification retry 都能出现在 timeline。
- audit 能反查同一 trace 下的用户动作。
- 常规数据量下单 trace 查询 p95 小于 1 秒；达不到时必须分页或启用 `trace_events`。
- 无权限用户不能通过 trace 看到未授权 payload。

## 8. OP-11-WF-RELIABILITY：Workflow 运行可靠性

### 8.1 目标

让 Workflow 不只是能跑通，而是在失败、取消、重试、版本变更后仍然可解释、可恢复、可审计。

### 8.2 功能范围

P0 包含：

- Workflow run action history。
- Retry plan preview。
- Cancel propagation report。
- Definition diff summary。
- Node output summary。
- Workflow run search。
- Retry/cancel 幂等保护和并发状态机测试。

### 8.3 Action History

新增运行期动作记录，覆盖：

- `retry_requested`
- `retry_started`
- `retry_completed`
- `cancel_requested`
- `cancel_propagated`
- `cancel_completed`
- `approval_approved`
- `approval_rejected`
- `node_skipped`
- `node_output_summarized`

建议表：

```sql
workflow_run_actions(
  id,
  workflow_run_id,
  trace_id,
  action_type,
  actor_id,
  idempotency_key,
  status,
  request_json,
  result_json,
  created_at,
  completed_at
)
```

约束：

- `idempotency_key` 用于避免重复 retry/cancel。
- 同一个 run 的互斥动作必须有服务端并发保护。
- 每个 action 必须写 audit，并进入 Trace Center。

### 8.4 Retry Plan Preview

重试前必须展示影响范围：

| 内容 | 说明 |
| --- | --- |
| 起点节点 | 从哪个 failed/canceled node 开始 |
| 将重跑节点 | 需要重新执行的节点列表 |
| 将跳过节点 | 已成功且可复用结果的节点 |
| 风险提示 | 是否包含批量主机、高危模板、外部通知 |
| 输入差异 | 当前 definition 与执行 definition 是否有差异 |

接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `GET` | `/api/v1/workflows/runs/:id/retry-plan` | 返回重试计划 |
| `POST` | `/api/v1/workflows/runs/:id/retry` | 提交重试动作 |

验收：

- 失败后从失败节点重试不会重复已成功节点。
- 重试动作重复提交时不会重复创建生产执行。
- retry plan 和最终执行结果能对应。

### 8.5 Cancel Propagation Report

取消操作必须解释收敛状态：

- 哪些 workflow node 已取消。
- 哪些 task target 已取消。
- 哪些 target 已经发出 cancel 但 Agent 尚未确认。
- 哪些 target 已完成，无法取消。
- 哪些 notification 已跳过或已投递。

接口：

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| `POST` | `/api/v1/workflows/runs/:id/cancel` | 提交取消 |
| `GET` | `/api/v1/workflows/runs/:id/cancel-report` | 查看取消传播报告 |

验收：

- 取消运行中 task 节点后，task target 状态和 workflow node 状态最终一致。
- 对已经完成的节点不做伪取消，必须明确展示不可取消原因。
- cancel 进入 action history、audit 和 Trace Center。

### 8.6 Definition Diff Summary

Workflow run 必须能解释“执行时版本”和“当前版本”的差异：

- 节点新增、删除、修改。
- 边关系变化。
- 输入参数 schema 变化。
- risk 标记变化。
- notification / webhook / schedule 绑定变化。

首版只需要摘要，不要求完整可视化 diff。

### 8.7 Node Output Summary

节点输出摘要用于减少打开完整日志的成本：

- 成功节点：展示输出前 N 行、关键结果字段、耗时。
- 失败节点：展示 exit code、stderr 摘要、失败分类。
- 外部投递节点：展示目标 channel、投递状态、错误摘要。

规则：

- 不保存 secret 明文。
- 超长日志只保存摘要和日志引用。
- output summary 变化进入 Trace Center。

### 8.8 Workflow Run Search

新增或强化筛选：

- 状态：running、failed、succeeded、canceled。
- 触发源：manual、schedule、webhook。
- 时间范围。
- 发起人。
- workflow key / version。
- traceId。
- 关键字。

列表性能要求：

- 默认分页。
- 大表查询必须走索引或限定时间范围。
- 不允许前端一次性加载全部 run 再过滤。

## 9. OP-11-UI-SMOKE：Playwright UI Smoke

### 9.1 目标

把 V1.0 的“构建通过”升级为“关键页面可打开、可截图、可回归”。

### 9.2 覆盖页面

P0 覆盖：

- Dashboard。
- Workflow list / Workflow run detail。
- Agents。
- Tasks 或 Task run detail。
- Metrics。
- Alerts。
- Notifications。
- Audit。
- Webhooks。
- Trace Center。

P1 覆盖：

- Incident detail。
- Agent diagnostics diff。
- Security Review。
- UI Smoke Evidence 页面。

### 9.3 测试策略

Playwright smoke 不是完整 E2E 业务测试，它只验证关键 UI 可用性：

- 页面能打开。
- 主要标题、表格、状态组件可见。
- 无 JS runtime error。
- 无明显空白页面。
- 窄屏下关键文本不溢出。
- icon button 有可识别 label 或 tooltip。
- 关键表单控件有 label 或 aria-label。

### 9.4 Seed 数据

必须提供稳定 seed：

- 1 个在线 Agent、1 个 stale Agent。
- 1 个成功 task run、1 个失败 task run。
- 1 个成功 workflow run、1 个失败 workflow run。
- 1 个 webhook event。
- 1 个 notification delivery。
- 1 条 audit log。
- 1 条 alert 或 incident 投影。
- 1 条 trace。

Seed 方式优先级：

1. API setup，保证贴近真实系统。
2. 测试专用 fixture。
3. 静态 mock，仅允许用于仍未接入后端的 P2 页面。

### 9.5 Artifact 规范

CI 或本地 smoke 应输出：

```text
artifacts/ui-smoke/
  latest/
    desktop/
    narrow/
    traces/
    summary.json
```

`summary.json` 建议包含：

- run id。
- commit hash。
- startedAt / completedAt。
- viewport。
- page name。
- status。
- screenshot path。
- trace path。
- error summary。

### 9.6 发布证据

V1.1 release evidence 必须记录：

- Playwright 执行命令。
- 通过页面列表。
- desktop/narrow 截图路径。
- 失败页面和修复结论。
- CI artifact 路径。

## 10. P1：Agent Fleet 规模化治理

P1 只有在 P0 完成后推进，完成不了则进入 V1.2 或 V1.3。

### 10.1 功能范围

- Agent version distribution。
- Agent stale reason。
- Host Group 动态规则草案。
- 批量维护窗口 preview。
- Agent diagnostics diff。

### 10.2 Agent Stale Reason

将 Agent 异常原因从单一离线状态细分为：

- 心跳超时。
- token 失效。
- Agent 被禁用。
- 网络不可达。
- 版本过旧。
- 最近执行失败率过高。

### 10.3 验收

- 运维人员能看出哪些 Agent 需要升级。
- 批量操作前能看到影响范围。
- diagnostics diff 不泄露敏感环境变量。

## 11. P1：告警与 Incident 治理

### 11.1 功能范围

- Incident 手动创建、合并、关闭。
- Alert group 到 incident 的关联与解除。
- Routing hit explanation。
- Noise report。
- Maintenance impact report。

### 11.2 设计边界

V1.1 不完整实现 V1.2 的 Incident 生命周期。V1.1 只补最影响可解释性的能力：

- 告警为什么投递到这个 channel。
- 告警为什么被 suppression 命中。
- 维护窗口期间哪些告警被抑制。
- 手工 incident 是否能关联已有 alert group。

### 11.3 验收

- 值班人员能从 incident 页面看到告警来源、处理记录、通知投递、关联任务。
- routing/suppression 命中原因可解释、可审计。
- 如果 P1 未纳入 V1.1，则这些验收不作为发布阻塞。

## 12. P1：安全与合规加固

### 12.1 功能范围

- Secret rotation history。
- 敏感配置变更二次确认。
- 权限变更影响预览。
- Audit export hash。
- 高危 workflow/template 操作 audit action 细化。

### 12.2 Secret Rotation History

记录：

- secret key id。
- rotation time。
- operator。
- old version / new version。
- related workflow/template/channel。

禁止：

- 回显明文。
- 在 audit metadata 中保存 secret value。
- 在 trace metadata 中展开 secret。

### 12.3 Permission Diff

权限变更前展示：

- 新增权限。
- 移除权限。
- 受影响角色。
- 受影响用户数量。
- 是否包含高危执行能力。

### 12.4 Audit Export Hash

导出审计文件时生成：

- hash algorithm。
- hash value。
- export time。
- exporter。
- filter condition。

验收：

- 导出的 audit 文件可校验完整性。
- 导出行为本身写入 audit。

## 13. 数据模型规划

### 13.1 P0 建议新增表

| 表 | 用途 | 是否必须 |
| --- | --- | --- |
| `workflow_run_actions` | retry/cancel/action history | 必须 |
| `ui_smoke_runs` | UI smoke 元数据，可只存 artifact 引用 | 可选，先文档化 artifact 也可 |
| `trace_events` | Trace timeline 缓存 | 条件必须，性能不达标时启用 |

### 13.2 P1 建议新增表

| 表 | 用途 |
| --- | --- |
| `incident_alert_groups` | incident 与 alert group 多对多关系 |
| `secret_rotation_events` | secret 轮换历史 |
| `agent_diagnostic_snapshots` | diagnostics diff 基础数据 |

### 13.3 字段扩展建议

可能需要给现有表补字段：

- `trace_id`：workflow run、task run、task target、webhook event、notification delivery、audit log。
- `idempotency_key`：workflow action 请求。
- `summary_json`：workflow node output summary。
- `retention_policy_key`：事件类表可选。

### 13.4 迁移要求

- 所有新增表必须有 up/down 迁移。
- 新字段必须允许为空或提供默认值。
- 索引必须对应实际查询条件。
- 大表索引新增必须评估写入影响。
- 任何 retention/delete 功能必须先提供 dry-run、影响行数和 audit。
- 迁移说明必须写入 V1.1 release evidence。

## 14. API 规划

### 14.1 P0 API

| 方法 | 路径 | Epic | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/traces/:id` | TRACE | trace summary |
| `GET` | `/api/v1/traces/:id/events` | TRACE | trace timeline 分页 |
| `GET` | `/api/v1/traces/lookup` | TRACE | entity 到 trace 查找 |
| `GET` | `/api/v1/workflows/runs/:id/retry-plan` | WF | retry plan preview |
| `POST` | `/api/v1/workflows/runs/:id/retry` | WF | 提交 retry |
| `POST` | `/api/v1/workflows/runs/:id/cancel` | WF | 提交 cancel |
| `GET` | `/api/v1/workflows/runs/:id/cancel-report` | WF | cancel propagation |
| `GET` | `/api/v1/workflows/runs/:id/action-history` | WF | action history |
| `GET` | `/api/v1/workflows/runs/:id/definition-diff` | WF | definition diff summary |

### 14.2 P1 API

| 方法 | 路径 | Epic | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/agents/version-distribution` | Agent | Agent 版本分布 |
| `GET` | `/api/v1/agents/:id/diagnostics/diff` | Agent | diagnostics diff |
| `GET` | `/api/v1/alerts/:id/routing-explanation` | Incident | routing/suppression explanation |
| `GET` | `/api/v1/alerts/noise-report` | Incident | 噪声报表 |
| `GET` | `/api/v1/incidents/:id/timeline` | Incident | incident timeline |
| `POST` | `/api/v1/incidents/:id/alert-groups` | Incident | 关联 alert group |
| `GET` | `/api/v1/security/permissions/diff` | Security | 权限差异 |
| `GET` | `/api/v1/security/secrets/:id/rotation-history` | Security | secret 轮换历史 |

### 14.3 API 约束

- 所有新增 API 必须进入 OpenAPI。
- router 与 OpenAPI diff 必须通过。
- 执行类 API 必须有幂等策略。
- 查询类 API 默认分页或限制时间范围。
- API 响应中的敏感字段必须统一脱敏。
- breaking/non-breaking 变更必须在 release evidence 标注。

## 15. 前端信息架构

### 15.1 新增页面

| 页面 | 路由建议 | 优先级 |
| --- | --- | --- |
| Trace Center | `/traces`、`/traces/:id` | P0 |
| Workflow Run Action Panel | Workflow run detail 内嵌 | P0 |
| UI Smoke Evidence | `/system/ui-smoke` 或文档 artifact | P0/P1 |
| Agent Fleet Overview | `/agents/fleet` | P1 |
| Incident Detail 增强 | `/incidents/:id` | P1 |
| Security Review | `/security/review` | P1 |

### 15.2 页面设计要求

- 控制台页面保持高信息密度、克制视觉风格。
- 不做营销页、装饰 hero、无关插画。
- 表格、timeline、详情面板必须有稳定尺寸和响应式约束。
- icon button 使用现有图标库或 lucide icons，并配 tooltip。
- 关键操作有 loading、empty、error、permission denied 状态。
- 长 trace id、host name、workflow key、error summary 必须可换行或截断，不允许溢出。

### 15.3 导航调整

建议在主导航或系统工具区加入：

- Trace Center。
- Workflow Runs。
- UI Smoke Evidence。

P1 页面可先挂在对应模块内，避免导航膨胀。

## 16. 权限与审计

### 16.1 权限建议

新增权限应尽量归入现有权限组，避免权限爆炸。

| 权限 | 说明 |
| --- | --- |
| `traces:read` | 查看 trace 基础 timeline |
| `traces:read_sensitive` | 查看脱敏前或更详细 metadata，默认不授予 |
| `workflow_runs:retry` | 重试 workflow run |
| `workflow_runs:cancel` | 取消 workflow run |
| `workflow_runs:read_actions` | 查看 action history |
| `ui_smoke:read` | 查看 UI smoke 结果 |
| `security:review` | 查看权限 diff、secret rotation history |

### 16.2 Audit 规则

必须写 audit 的动作：

- 查看或导出敏感 trace metadata。
- retry workflow。
- cancel workflow。
- 修改 secret。
- 导出 audit。
- 权限变更 preview 和提交。
- incident 合并、关闭、关联 alert group。

### 16.3 隐私和脱敏

- Trace metadata 默认脱敏。
- Task log summary 不保存 secret。
- webhook header 默认只展示 allowlist 字段。
- Notification delivery 不展示 token、webhook secret。
- Audit export 标明脱敏状态。

## 17. 测试策略

### 17.1 后端测试

必须覆盖：

- Trace 聚合 manual/schedule/webhook 三条路径。
- 无权限 trace 查询。
- retry plan 的 skip/re-run 计算。
- retry 幂等。
- cancel 并发与最终状态一致。
- definition diff summary。
- migration up/down。

### 17.2 前端测试

必须覆盖：

- Trace Center 有数据、空状态、无权限、错误状态。
- Workflow run detail 的 retry plan、cancel report、action history。
- desktop/narrow 两个 viewport 的 Playwright smoke。
- 关键页面无 runtime error。
- 关键按钮可键盘聚焦，有 label 或 tooltip。

### 17.3 集成测试

建议覆盖：

- webhook 触发 workflow，失败后进入 Trace Center，重试后状态更新。
- schedule 触发 workflow，取消后 cancel report 可见。
- notification retry 进入 trace timeline。
- audit 能按 traceId 反查动作。

### 17.4 发布前命令

V1.1 发布前必须通过：

```bash
go test ./...
go vet ./...
npm run build
npm run test:ui-smoke
```

并补充：

- OpenAPI/router diff。
- migration pair check。
- full-stack Compose smoke。
- Markdown local links check。
- `git diff --check`。

## 18. 发布证据与文档

### 18.1 必须更新的文档

- `README.md`
- `README.en.md`
- `CHANGELOG.md`
- `CHANGELOG.en.md`
- `docs/version/v1.1/product-development-design.md`
- V1.1 release evidence，发布时新增
- OpenAPI 文档

### 18.2 Release Evidence 内容

V1.1 release evidence 至少包含：

- 版本号、commit、构建时间。
- P0 完成清单。
- P1 纳入范围和延期说明。
- 测试命令和结果。
- Playwright screenshot artifact。
- Trace Center manual/schedule/webhook 验收记录。
- Workflow retry/cancel 验收记录。
- migration pair check。
- OpenAPI/router diff。
- 兼容性和已知问题。
- 路线图偏差说明。

## 19. 里程碑计划

建议 5 周完成 V1.1。

### 第 1 周：Trace 基础链路

目标：

- 定义 trace id 传播规则。
- 完成 trace 聚合接口最小版本。
- Workflow、Task、Webhook、Notification、Audit 写入或关联 traceId。
- 前端 Trace Center 页面骨架。

交付：

- `GET /api/v1/traces/:id`。
- Trace Center 可展示 manual workflow 基础 timeline。
- OpenAPI 更新。

### 第 2 周：Workflow 可靠性

目标：

- workflow_run_actions。
- retry plan preview。
- cancel propagation report。
- action history。
- definition diff summary。

交付：

- retry/cancel 幂等测试。
- Workflow run detail 操作面板。
- trace timeline 展示 retry/cancel。

### 第 3 周：UI Smoke 和发布证据

目标：

- Playwright 配置。
- seed 数据。
- desktop/narrow viewport。
- screenshot/trace artifact。

交付：

- `npm run test:ui-smoke`。
- 关键页面截图。
- release evidence 模板。

### 第 4 周：P0 收敛与 P1 候选

目标：

- 修复 P0 漏洞。
- 性能和权限校验。
- 只有 P0 已闭环时推进 P1。

可选交付：

- Agent version distribution。
- routing explanation。
- secret rotation history。

### 第 5 周：发布收口

目标：

- 全量测试。
- 文档更新。
- release evidence 完成。
- P1 完成/延期清单。

交付：

- V1.1 发布候选。
- CHANGELOG。
- 发布验收记录。

## 20. 依赖关系

### 20.1 Epic DAG

```text
V1.0 release baseline
  -> Trace id propagation
    -> Trace Center API
    -> Workflow action timeline
    -> UI smoke trace evidence
  -> Workflow run state machine
    -> Retry plan
    -> Cancel report
    -> Action history
  -> Stable seed data
    -> Playwright UI smoke
    -> Release evidence
```

### 20.2 阻塞规则

- Trace id 传播未完成时，不进入 Trace Center 完整页面验收。
- retry/cancel 幂等未完成时，不允许开放 UI 操作。
- Playwright seed 不稳定时，不允许把 UI smoke 设为发布通过。
- P0 任一项未完成时，不推进 P1 生产功能。

## 21. 风险与修复

| 风险 | 影响 | 修复措施 |
| --- | --- | --- |
| Trace 聚合查询慢 | 排障入口不可用 | p95 超 1 秒时启用分页或 `trace_events` 缓存 |
| Trace 泄露敏感数据 | 安全事故 | 复用脱敏策略，敏感 metadata 权限隔离，查看写 audit |
| retry 重复执行生产命令 | 生产风险 | idempotency key、服务端锁、状态机测试 |
| cancel 状态不一致 | 用户误判任务已停止 | cancel report 展示收敛状态，不伪造最终态 |
| UI smoke flaky | CI 不可信 | deterministic seed、固定 viewport、禁动画、稳定 selector |
| P1 范围膨胀 | V1.1 延期 | P1 未完成不阻塞，P0 独立发布 |
| 新增迁移破坏 V1.0 数据 | 升级失败 | up/down、默认值、旧数据样例、row-count 校验 |
| 权限模型膨胀 | 角色难维护 | 新权限归组，权限 diff，新增权限必须映射角色 |
| Agent/Incident P1 半成品 | 发布质量下降 | 未达到验收则延期到 V1.2/V1.3，不挂到主导航 |

## 22. 回退与降级

### 22.1 Trace Center

- 如果聚合性能不足，先分页 timeline。
- 如果仍不足，启用 `trace_events` 缓存。
- 如果缓存写入异常，回退到按需聚合，并在 UI 标记 partial timeline。

### 22.2 Workflow Retry/Cancel

- 如果 retry plan 无法完整解释复杂 DAG，先只允许从 failed node 重试。
- 如果 cancel 无法保证 Agent 侧即时停止，UI 必须展示 cancel requested / converging，而不是 canceled。
- 如果幂等保护未完成，禁止开放 UI retry/cancel 操作。

### 22.3 UI Smoke

- 如果 CI 环境不稳定，保留本地命令和 artifact，但必须说明阻塞原因。
- 如果某页面缺少 seed，页面可先以 empty state 验收，但必须记录缺口。

## 23. 硬停止条件

以下任一条件成立，V1.1 不得发布：

- Trace Center 无法覆盖 manual/schedule/webhook 三条 P0 路径。
- Trace 查询会泄露用户无权访问的 payload、secret、header 或日志。
- Workflow retry/cancel 没有幂等保护。
- Workflow retry/cancel 没有 action history 和 audit。
- Playwright UI smoke 无法在 desktop 和 narrow viewport 通过。
- 新增迁移没有 down 文件。
- OpenAPI/router diff 失败。
- `go test ./...`、`go vet ./...`、`npm run build` 任一失败。
- P1 未完成却被写成发布已完成。
- README、CHANGELOG、release evidence 与实际功能状态不一致。

## 24. V1.1 完成定义

V1.1 只有同时满足以下条件才算完成：

- P0 三个 Epic 全部通过验收。
- Trace Center 能串起触发、执行、通知、审计。
- Workflow retry/cancel 可预览、可审计、可追踪、可幂等。
- UI smoke 产生可引用截图证据。
- 所有新增 API 有 OpenAPI。
- 所有新增迁移成对。
- 所有新增敏感能力有权限和 audit。
- 发布证据记录测试结果、截图、已知问题和路线图偏差。
- P1 状态明确：完成、延期或取消，不能模糊描述。

## 25. V1.2 衔接

V1.1 完成后，V1.2 应优先承接：

- Incident 完整生命周期。
- 告警噪声治理。
- 变更风险控制。
- 执行审批策略。
- 运维运营报表。

V1.1 不应该提前实现 V1.2 的完整治理模型，只需要为它打好 trace、action history、audit、UI smoke 基础。

## 26. 最终结论

V1.1 的正确做法不是扩大业务边界，而是把 V1.0 的主链路做扎实。发布承诺应严格绑定三个 P0：

1. Trace Center。
2. Workflow 运行可靠性。
3. Playwright UI Smoke。

Agent Fleet、Incident 治理、安全合规可以做，但只能作为 P1 加固项。只要 P0 端到端成立，V1.1 就能为 V1.2/V1.3 的治理和规模化能力提供可靠基础；如果 P0 不成立，继续堆功能只会增加排障和发布风险。

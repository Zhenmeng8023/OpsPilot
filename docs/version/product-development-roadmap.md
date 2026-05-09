# OpsPilot 中长期软件开发设计规划

生成日期：2026-05-09

## 1. 当前主业务边界

OpsPilot V1.0 的主业务是自动化运维控制平面，已经覆盖 Agent 接入、脚本模板、任务执行、实时日志、Cron/Webhook/Manual 触发、Workflow 编排、Metrics/Alerts/Notifications、Incident 投影、RBAC 和 Audit。

未来版本可以扩展能力，但必须围绕这条主业务链路：

```text
外部触发 / 人工触发
  -> Workflow 编排
  -> Agent / Host 执行
  -> 日志 / 指标 / 事件采集
  -> 告警 / 事件 / 通知
  -> 审计 / 复盘 / 治理
```

V1.x 的任务是把这条链路打磨成生产可用、可排障、可治理、可规模化运行的平台。

V2.0 才适合引入新的主业务：从“自动化运维控制平面”升级为“运维服务交付与治理平台”。这个新主业务仍然强关联当前主业务，但不应塞进 V1.1/V1.2 这类小版本。

## 2. 长期产品愿景

OpsPilot 的长期形态可以分为三层：

1. 自动化执行层：Agent、脚本、任务、Workflow、调度、Webhook。
2. 生产运维治理层：Trace、Incident、告警治理、审计、安全、变更风险控制。
3. 运维服务交付层：服务目录、标准作业、自助申请、审批、执行、回执、SLO 和复盘。

V1.0 已经完成第一层和部分第二层。V1.1 到 V1.3 应该补强第二层。V2.0 再系统设计第三层。

## 3. 版本路线图总览

| 版本 | 定位 | 主线 | 不做 |
| --- | --- | --- | --- |
| V1.1 | 生产化加固 | Trace Center、Workflow 可靠性、Playwright UI smoke、Agent Fleet、Incident 治理、安全合规 | 新主业务、多租户、AI 自动修复 |
| V1.2 | 运维治理增强 | Incident 生命周期、告警噪声治理、变更风险控制、执行策略、报表 | 服务目录、商业化租户 |
| V1.3 | 规模化与扩展边界 | 大规模 Agent 管理、插件化节点、集成中心、外部系统连接器 | 微服务化重构、复杂计费 |
| V2.0 | 新主业务：运维服务交付平台 | 服务目录、标准作业、申请审批、SLO、回执、复盘、服务治理 | 继续用零散任务页承载服务交付 |
| V2.1+ | 平台化与企业能力 | 多空间/多团队、SSO/OIDC、策略中心、审计归档、开放生态 | 破坏 V2.0 服务模型 |

## 4. V1.1 生产化加固

V1.1 目标是让 V1.0 在真实值班中更好用，而不是增加新的业务模型。

P0：

- Trace Center：全局排障入口，串起 workflow、task、webhook、schedule、notification、audit、alert/incident。
- Workflow 运行可靠性：retry plan、cancel propagation、action history、definition diff、node output summary。
- Playwright UI smoke：关键页面桌面/窄屏截图、CI artifact、失败 trace。

P1：

- Agent version distribution、stale reason、diagnostics diff。
- Incident 手动合并/关闭、routing explanation、noise report。
- Secret rotation history、权限 diff、audit export hash。

验收重点：

- 值班人员能从一个 trace 还原一次故障处理链路。
- Workflow 失败恢复行为可预览、可审计。
- UI 不再只靠 build 证明可用。

详细设计见 `docs/version/v1.1/product-development-design.md`。

## 5. V1.2 运维治理增强

V1.2 目标是把“能执行”和“能看到”升级为“能治理”。它仍属于当前主业务，不引入服务目录这种新主业务。

### 5.1 Incident 生命周期

范围：

- Incident 状态机：`open -> acknowledged -> mitigating -> resolved -> closed`。
- 负责人、协作者、影响范围、根因分类、恢复时间。
- Incident timeline 支持手工备注、关联 workflow run、关联 alert group、关联 notification。
- Postmortem 草稿：自动汇总时间线、处理动作、通知记录和影响范围。

验收：

- 一次告警风暴能归并成一个 incident。
- incident close 前必须有处理结论或豁免原因。
- postmortem 草稿能从真实 timeline 生成。

### 5.2 告警噪声治理

范围：

- Alert rule health：规则触发频率、恢复率、误报标记、无人处理比例。
- Suppression simulation：新建抑制规则前预估近 7 天会抑制哪些告警。
- Routing dry-run：测试某条 alert 会投递到哪些 channel。
- Noise budget：按 host group / service / severity 设置噪声预算。

验收：

- 能识别最高噪声的前 10 条规则。
- 修改 routing/suppression 前能预览影响。

### 5.3 变更风险控制

范围：

- 高危 Workflow 标记：涉及批量主机、生产标签、高危命令、夜间窗口。
- Change window：执行前必须落在允许时间段。
- Approval policy：按 workflow risk、host group、operator role 匹配审批策略。
- Dry-run node：允许节点只解析目标和参数，不执行命令。

验收：

- 高危批量操作不能绕过审批。
- 审批记录、执行记录、取消记录可以完整审计。

### 5.4 运营报表

范围：

- Weekly Ops Report：任务成功率、平均执行时长、失败 TopN、告警噪声、incident MTTA/MTTR。
- Agent Health Report：在线率、版本分布、异常主机、诊断变化。
- Workflow Usage Report：最常用 workflow、失败最多节点、平均恢复时间。

验收：

- 报表可按时间范围导出。
- 报表数据来源可追溯到具体 run、alert、incident。

## 6. V1.3 规模化与扩展边界

V1.3 目标是支持更大规模和更多外部系统集成，同时仍以自动化运维主业务为中心。

### 6.1 Agent Fleet 规模化

范围：

- Agent enrollment profile：不同环境使用不同注册策略、默认标签和限制。
- Dynamic host groups：基于 tag、hostname、OS、version、region 的动态分组。
- Batch operation planner：批量操作分批、限速、失败暂停。
- Agent upgrade protocol 设计与灰度升级基础能力。

验收：

- 1000+ Agent 的列表、分组、批量操作设计不依赖前端一次性加载。
- 批量任务可以按批次追踪、暂停和恢复。

### 6.2 Workflow 节点扩展机制

范围：

- Node registry：内置节点与扩展节点统一描述。
- Node schema：输入输出 schema、UI form schema、权限要求。
- Connector node：HTTP、SQL read-only、message queue、GitHub/GitLab webhook callback。
- Node sandbox policy：限制可访问 secret、网络目标和执行超时。

验收：

- 新增一种节点不需要修改 Workflow runtime 核心状态机。
- 节点权限、secret 使用、审计可被统一治理。

### 6.3 Integration Center

范围：

- 集成对象：GitHub/GitLab、Prometheus-compatible webhook、Slack/Teams/飞书/钉钉、Jira/Linear 类工单系统。
- 每个集成提供 health check、secret rotation、test event、delivery history。
- Webhook source 与 notification channel 合并进入统一集成视图，但底层数据模型可保持兼容。

验收：

- 外部系统接入状态可见。
- 集成失败能进入 Trace Center。

## 7. V2.0 新主业务：运维服务交付平台

V2.0 是新主业务版本，不应放进小版本。它要把 OpsPilot 从“运维人员使用的自动化控制台”扩展成“业务团队可自助使用的运维服务交付平台”。

新主业务定义：

```text
服务目录
  -> 用户申请
  -> 风险评估 / 审批
  -> Workflow 执行
  -> 结果回执
  -> SLO / 满意度 / 复盘
```

它和现有主业务强关联：服务目录里的每个服务项最终仍调用 Workflow、Agent、Notification、Audit、Incident。但用户入口、权限模型、数据模型和成功标准都不同，所以必须作为 V2.0 级别设计。

### 7.1 Service Catalog

范围：

- Service item：例如重启服务、扩容磁盘、发布检查、日志拉取、证书更新、权限开通。
- Request form schema：面向申请人的表单，不暴露底层 workflow 复杂性。
- Fulfillment binding：服务项绑定 workflow version 和输入映射。
- Risk profile：服务风险、审批策略、可申请范围、SLO。

验收：

- 非管理员用户可以从服务目录提交标准运维申请。
- 运维人员可以把成熟 workflow 包装成服务项。

### 7.2 Request Center

范围：

- Request 状态机：`draft -> submitted -> approved/rejected -> running -> fulfilled/failed/canceled`。
- 申请人视图与运维视图分离。
- 每个 request 关联审批、workflow run、通知、审计、回执。
- 支持撤回、补充参数、重新提交。

验收：

- 申请人不需要理解 Agent/Workflow 细节也能查看进度。
- 运维人员能从 request 追踪到底层执行链路。

### 7.3 服务交付 SLO

范围：

- 每类服务项定义响应 SLO 和完成 SLO。
- Request 超时提醒、升级通知。
- 服务项维度统计成功率、耗时、失败原因、满意度。

验收：

- 能看出哪些标准服务交付慢、失败多、审批积压。

### 7.4 服务治理

范围：

- 服务项版本管理。
- 服务项下线/灰度。
- 服务项使用权限。
- 服务项变更审计。
- 服务项到 workflow 的依赖影响分析。

验收：

- 修改底层 workflow 前能看到影响哪些服务项。
- 服务项变更可回溯。

## 8. V2.1+ 企业平台能力

V2.1 以后再考虑更重的企业能力：

- Workspace / Team：团队隔离、资源归属、权限边界。
- SSO/OIDC：企业身份接入。
- Policy Center：统一定义审批、执行窗口、风险规则、secret 使用规则。
- Audit Archive：长期归档、签名、导出、保全。
- External Metrics Adapter：Prometheus、VictoriaMetrics、ClickHouse 等外部存储。
- HA / Scale：API 多副本、调度锁、Agent 长连接或消息队列演进。

这些能力必须服务于 V2.0 的服务交付模型，不能单独堆成架构复杂度。

## 9. 不建议做的方向

短中期不建议投入：

- AI 自动修复作为核心卖点。可以先做总结和建议，不要让 AI 直接执行生产修复。
- 复杂商业化计费。当前产品还应先证明运维服务交付价值。
- 微服务化拆分。现阶段模块化单体更利于交付和调试。
- 大规模替换前端技术栈。应先补 UI smoke 和设计系统一致性。
- 完整低代码平台。OpsPilot 需要的是运维工作流和服务交付，不是通用低代码。

## 10. 推荐开发节奏

建议按下面顺序推进：

1. V1.1：先解决排障、可靠性和 UI 验收。
2. V1.2：再解决 incident、告警噪声和变更风险。
3. V1.3：扩展 Agent 规模和外部集成能力。
4. V2.0：正式设计并实现服务目录与申请交付主业务。
5. V2.1+：在服务交付模型稳定后补企业平台能力。

这个顺序的关键是避免过早进入 V2.0。没有 Trace、Incident、Workflow 可靠性和 Agent 规模化基础，服务目录只会变成一个漂亮入口，底层交付质量仍然不可控。

## 11. 近期可执行拆分

下一轮开发可以先拆成 6 个 Epic：

| Epic | 对应版本 | 目标 |
| --- | --- | --- |
| OP-11-TRACE | V1.1 | 建立跨模块 Trace Center |
| OP-11-WF-RELIABILITY | V1.1 | Workflow retry/cancel/action history 加固 |
| OP-11-UI-SMOKE | V1.1 | Playwright UI smoke 和截图证据 |
| OP-12-INCIDENT | V1.2 | Incident 生命周期和 postmortem |
| OP-12-RISK | V1.2 | 变更风险、审批策略和执行窗口 |
| OP-13-INTEGRATION | V1.3 | Integration Center 与节点扩展边界 |

每个 Epic 必须同步：

- 后端 API 和测试。
- 前端页面和 i18n。
- OpenAPI/router check。
- migration pair check。
- release evidence。
- README/CHANGELOG/版本设计文档。

## 12. 成功标准

V1.x 阶段成功标准：

- 值班人员能在 5 分钟内定位一次 workflow/webhook/schedule 触发失败的主要原因。
- 高危批量操作默认受审批、窗口、审计约束。
- 告警噪声明显可见，并能用 suppression/routing/incident 治理。
- 关键 UI 页面有自动截图证据。
- Agent Fleet 能支撑分组、版本、异常原因和批量计划。

V2.0 阶段成功标准：

- 普通业务用户可以通过服务目录提交运维请求。
- 运维团队可以把成熟 workflow 产品化为服务项。
- 每个服务请求都能追踪到审批、执行、通知、审计和回执。
- 服务项有 SLO、使用统计和变更治理。

## 13. 策略漏洞复盘与修复措施

本节用于回答“这个路线图是否足够可靠”。结论是：不能声称绝对 100%，但可以通过明确漏洞、修复措施和验收证据，把策略提升到事实上的工程可执行状态。

### 13.1 第一轮漏洞

| 漏洞 | 风险 | 修复措施 |
| --- | --- | --- |
| V2.0 新主业务太晚才验证 | 到 V2.0 才发现服务目录不适合目标用户，返工成本高 | V1.3 只做 discovery prototype：用文档和只读 mock 验证服务目录信息架构，不进入生产功能，不污染 V1.x 数据模型 |
| Trace Center 可能拖慢核心查询 | 跨 workflow/task/webhook/audit/notification 多表聚合会引入慢查询 | V1.1 先做按需聚合和必要索引；`trace_events` 只作为可选缓存表，必须有 retention 和重建策略 |
| V1.1 范围可能过大 | Trace、Workflow 可靠性、UI smoke、Agent、Incident、安全同时推进会影响交付 | V1.1 P0 只允许 Trace、Workflow 可靠性、UI smoke；Agent/Incident/Security 作为 P1，未完成不得阻塞 V1.1 |
| Integration Center 扩大安全攻击面 | 外部 connector、webhook、HTTP 节点可能泄露 secret 或访问内网敏感地址 | V1.3 必须先实现 connector sandbox、egress allowlist、secret scope、delivery redaction，再开放通用 connector |
| Agent 规模化早于 HA | 1000+ Agent 目标可能让单体调度、列表和 DB 压力提前暴露 | V1.3 必须补分页游标、批量操作分批、调度互斥锁、关键索引和压测脚本；真正 HA 留到 V2.1+ |
| 多团队/空间放到 V2.1 可能导致 V2.0 返工 | 服务目录天然有团队归属，如果完全不预留会重改权限和资源模型 | V1.2 起新增资源 ownership 元数据设计，V2.0 使用 owner/team 字段，但强隔离和计费仍推迟到 V2.1+ |
| Incident 和 Service Request 容易混淆 | incident 是故障处理，request 是服务交付；混成一个模型会导致状态机失控 | V2.0 明确新增 Request Center，不复用 Incident 状态机，只允许通过 trace/timeline 关联 |
| UI smoke 容易 flaky | E2E UI 截图依赖时间、数据和动画，CI 可能假失败 | V1.1 必须使用 deterministic seed、固定 viewport、禁用非必要动画、稳定 selector、失败截图和 trace artifact |
| 外部指标库推迟可能影响 Metrics 扩展 | MySQL metrics rollup 长期可能无法支撑大规模指标 | V1.3 只设计 adapter interface 和导出边界；V2.1 再接 Prometheus/VictoriaMetrics/ClickHouse |
| 缺少产品反馈闭环 | 路线图可能工程上合理，但没有验证真实使用价值 | V1.2 增加本地 usage report：workflow 使用率、失败恢复次数、incident 关闭耗时、告警噪声趋势，不上传外部服务 |

### 13.2 第二轮修正后的版本边界

- V1.1 只承诺 P0：Trace Center、Workflow 可靠性、Playwright UI smoke。
- V1.2 承诺治理：Incident lifecycle、告警噪声、变更风险、运营报表。
- V1.3 承诺扩展边界：Agent scale、node registry、Integration Center，但必须先完成 connector 安全边界。
- V2.0 承诺新主业务：Service Catalog、Request Center、SLO、服务治理。
- V2.1+ 承诺企业能力：workspace/team 强隔离、SSO/OIDC、Policy Center、外部指标库、HA。

任何版本如果出现范围膨胀，优先保留该版本的 P0 主线，把 P1/P2 延后，不允许为了多做功能牺牲发布证据和回归检查。

### 13.3 事实信心门槛

路线图只有在下面证据持续成立时，才可以认为“事实上的高信心”：

- 每个版本都有明确 P0/P1/P2，P0 可独立发布。
- 每个新增主能力都有 API、数据模型、前端、OpenAPI、迁移、测试、发布证据。
- 每个版本都保留 rollback 或降级方案；无法回滚的数据变更必须在迁移说明里明示。
- 所有 connector、secret、审批、批量执行能力都有审计和权限边界。
- UI smoke、Compose smoke、OpenAPI/router check、migration pair check、Go test/vet、frontend build 都是发布门槛。
- 版本完成后必须更新 README、CHANGELOG、Release Evidence 和对应版本设计文档。

### 13.4 第二轮漏洞

| 漏洞 | 风险 | 修复措施 |
| --- | --- | --- |
| V1.1 文档存在 P1 内容 | 团队可能把 Agent/Incident/Security P1 当成 V1.1 必须完成，导致范围膨胀 | V1.1 设计文档新增范围保护：V1.1 发布承诺只绑定 Trace、Workflow 可靠性、UI smoke 三个 P0 |
| V2.0 discovery 没有量化 go/no-go | 服务目录可能停留在概念上，无法判断是否值得进入 V2.0 | V1.3 discovery 必须验证至少 5 个真实标准运维服务项，且每个都能映射到现有 workflow、审批、通知、审计链路 |
| Trace Center 缺少性能目标 | 聚合接口可能能跑通但排障体验不可用 | V1.1 设计目标：单 trace 查询在常规数据量下 p95 小于 1 秒；超出时必须引入 trace_events 缓存或分页 timeline |
| Retention 策略仍偏抽象 | trace、audit、metrics、notification 数据增长可能互相挤压 | V1.1 起每个新增事件表必须声明 retention；默认先保留可配置项，不硬编码删除周期 |
| Ownership 预留不够具体 | V2.0 服务目录引入团队归属时可能重做资源表 | V1.2/V1.3 新资源优先预留 nullable `owner_type`、`owner_id` 或 `team_key`，但不启用强隔离 |
| 压测目标没有进入发布门槛 | 1000+ Agent 目标可能只停留在设计口号 | V1.3 必须提供 Agent list、heartbeat、batch planner 的压测脚本和结果记录，达不到目标则不得声明规模化完成 |

### 13.5 V2.0 Go/No-Go 门槛

V2.0 只有在 V1.3 完成以下 discovery 证据后才进入正式开发：

- 至少梳理 5 个标准运维服务项：例如重启服务、日志拉取、证书更新、发布前检查、磁盘扩容。
- 每个服务项都能绑定一个现有或计划中的 workflow version。
- 每个服务项都有申请人、审批人、执行人、可申请范围和失败回退说明。
- 至少 3 个服务项能复用 V1.x 的 Trace、Audit、Notification、Incident 能力。
- 服务目录 mock 不新增生产数据表，只输出信息架构、表单 schema 和状态机设计。

如果以上证据不成立，V2.0 不进入服务目录实现，继续在 V1.x 强化自动化运维控制平面。

### 13.6 第三轮漏洞

| 漏洞 | 风险 | 修复措施 |
| --- | --- | --- |
| V1.1 发布门槛仍引用 P1 Incident/routing/suppression | 范围保护失效，P1 被误当成阻塞项 | V1.1 发布门槛改为条件门槛：只有纳入 P1 改动时才要求 Incident/routing/suppression 可解释 |
| V1.1 交付节奏暗示第 4 周必须做 P1 | 迭代计划推动范围膨胀 | 第 4 周改为 P1 候选项，只在 P0 已闭环时推进；第 5 周优先发布验收 |
| Agent 信任链没有被显式纳入长期策略 | Agent 自动升级、批量执行、诊断上报可能被伪造或滥用 | V1.3 Agent scale 前必须设计 Agent identity、token rotation、version attestation、升级包签名校验；未完成前不得开放自动升级 |
| 数据备份与灾难恢复未进入门槛 | migration、retention、audit archive 可能造成不可恢复的数据损失 | V1.2 起所有 retention/归档功能必须提供 dry-run、影响行数、审计记录和恢复说明；V2.1 HA 前补备份/恢复演练 |
| API 兼容策略不明确 | V2.0 服务目录和 V1.x API 演进可能破坏现有客户端或 Agent | 从 V1.1 起 OpenAPI 变更需标注 breaking/non-breaking；Agent API 需要兼容窗口和最低支持版本 |
| 权限模型可能膨胀 | Trace、Incident、Service Catalog、Integration Center 增加大量 permission，角色难以维护 | V1.2 起做 permission grouping 和权限差异预览；新增权限必须映射到现有角色或明确迁移策略 |
| 审计与隐私边界未充分说明 | Trace Center 聚合 payload、headers、日志、通知可能扩大敏感信息暴露 | Trace Center 默认复用现有脱敏策略；payload/header/log 默认折叠并按权限展示；导出必须带审计和脱敏标记 |
| 运维服务交付可能绕过现有审批 | Service Catalog 包装 workflow 后，申请入口可能降低高危操作门槛 | V2.0 request 必须继承 workflow risk、approval policy、change window 和 target scope，不允许服务项降低底层安全级别 |

### 13.7 硬停止条件

以下任一条件成立时，对应版本不得发布：

- P0 主线没有端到端验收证据。
- 新增执行能力没有审计记录。
- 新增 secret 或 connector 能力没有脱敏、scope 和 rotation 策略。
- 新增批量执行能力没有 preview、分批、取消和失败收敛说明。
- 新增数据保留或删除能力没有 dry-run 和审计。
- 新增 Agent 自动升级能力没有签名校验和回滚策略。
- 新增 Service Catalog 能力降低了底层 workflow 的审批或执行窗口要求。
- OpenAPI/router diff、migration pair check、Go test/vet、frontend build 或 UI smoke 任一发布门槛失败。

### 13.8 第四轮漏洞

| 漏洞 | 风险 | 修复措施 |
| --- | --- | --- |
| 没有资源容量约束 | 路线图正确但开发容量不足，导致多条主线半成品并行 | 每个版本只能有 1 条 P0 主线组；V1.1 的 P0 是 Trace + Workflow reliability + UI smoke，P1 只能在 P0 完成后进入 |
| 版本依赖没有显式 DAG | V1.2/V1.3 可能依赖尚未完成的 Trace、Incident 或 ownership 基础 | 每个 Epic 开工前必须写依赖清单；依赖未完成时只能做设计，不进入实现 |
| 供应链安全未覆盖 | Playwright、connector SDK、前端依赖或 Actions 升级可能带来供应链风险 | V1.1 起新增依赖必须记录用途；CI 保留 npm audit/go vulncheck 预留位，严重漏洞不得进入发布 |
| 幂等与并发未进入策略 | retry、cancel、batch planner、request fulfillment 可能重复执行生产命令 | 所有执行类 API 必须设计 idempotency key 或服务端去重；Workflow retry 和批量执行必须有并发锁与状态机测试 |
| 跨平台 Agent 风险不足 | Windows/Linux shell、路径、编码、权限模型差异可能导致执行行为不一致 | Agent 相关版本必须维护 Windows/Linux 兼容矩阵；高危命令拦截、日志脱敏、工作目录隔离要双平台验收 |
| UI 可访问性未写入门槛 | 运维控制台高密度页面可能键盘不可用、文本溢出或对比度不足 | UI smoke 增加基本 accessibility 检查：可聚焦、label/aria-label、文本不溢出、窄屏可操作 |
| 数据迁移演练不够 | V2.0 服务目录和 ownership 字段可能在真实数据上迁移失败 | V1.2 起重大迁移必须提供旧数据样例、升级路径、回滚路径和 row-count 校验 |
| 配置与 feature flag 缺失 | 新能力一旦上线无法快速关闭 | P0 新能力默认提供配置开关或降级路径，尤其是 Trace cache、connector、Agent upgrade、Service Catalog |
| 文档与实现可能再次漂移 | 路线图长期存在，后续实现可能偏离但文档不更新 | 每个版本 release evidence 必须包含“路线图偏差”小节，说明完成、延期、取消和原因 |

### 13.9 执行前检查清单

任何 Epic 从设计进入实现前，必须回答：

- 这个 Epic 的 P0 成功标准是什么，是否可以独立发布。
- 依赖哪些已完成模块，依赖是否已有测试和 release evidence。
- 是否新增执行路径，若是，幂等、取消、并发锁和审计在哪里实现。
- 是否新增外部依赖，若是，供应链、license、漏洞和升级策略是什么。
- 是否新增或修改迁移，若是，旧数据样例、回滚路径、row-count 校验和 retention 策略是什么。
- 是否影响 Agent，若是，Windows/Linux 兼容矩阵和最低 Agent 版本是什么。
- 是否影响 UI，若是，desktop/narrow viewport、键盘、label、文本溢出和截图证据是什么。
- 是否可以通过配置开关降级。

## 14. 决策门与回退规则

每个版本进入开发前必须过一次设计门：

1. 是否仍围绕当前主业务，或是否应该提升为 V2.0 级新主业务。
2. 是否有明确用户、入口、状态机和验收证据。
3. 是否影响权限、审计、secret、批量执行或外部投递。
4. 是否需要新增迁移、索引、retention 或归档策略。
5. 是否能被 Trace Center 或 Release Evidence 观测。

回退规则：

- 如果 P0 超过一个迭代仍无法闭环，拆分版本，不继续堆 P1。
- 如果新增能力无法审计，不能进入发布分支。
- 如果新增 connector 无法限制 secret scope 和 egress，不能开放给用户配置。
- 如果某个 V1.x 功能开始改变用户入口、权限模型和业务对象，应升级到 V2.0 设计，不进入小版本。

## 15. 当前置信结论

经过四轮漏洞复盘和修复后，当前策略在以下范围内具备事实上的高信心：

- V1.1 到 V1.3 的方向和当前 V1.0 主业务强绑定。
- V2.0 新主业务与当前主业务有明确依赖关系，但不会污染 V1.x 小版本。
- 主要工程风险都有对应修复门槛：性能、范围、安全、数据增长、UI flaky、HA 时机、多团队预留、产品反馈、V2.0 discovery、Agent 信任链、灾备、API 兼容、权限膨胀、审计隐私、资源容量、供应链、幂等并发、跨平台 Agent、可访问性、迁移演练和配置降级。

仍不能声称数学意义上的 100%：

- 真实用户是否接受 V2.0 服务目录，需要 V1.3 discovery prototype 验证。
- 1000+ Agent 和高频指标规模，需要压测数据验证。
- Connector 生态是否值得做，需要实际集成需求验证。
- Agent 自动升级和外部 connector 的安全边界，需要实现前的威胁建模和验证。
- 真实容量约束会随团队规模变化，需要每个版本开工前重新校准。

因此，策略的最终信心不是来自一次文档判断，而是来自每个版本发布门槛和复盘证据持续成立。

### 15.1 停止继续循环的事实标准

只有同时满足下面条件，才允许停止“继续找漏洞”的循环：

- 已按范围、性能、安全、数据、交付、用户价值、兼容、迁移、供应链、Agent、UI、权限、审计、灾备、V2.0 新主业务边界逐类检查。
- 每一类风险都有至少一个修复措施、验收门槛或硬停止条件。
- 文档中不存在已知自相矛盾的发布门槛，例如 P1 被无条件写成 P0 阻塞项。
- 新增能力都有降级、回滚或延期路径；没有路径时必须被硬停止条件拦截。
- `git diff --check` 和 `git diff --cached --check` 通过。
- 当前结论明确区分“事实上的工程高信心”和“数学意义上的绝对 100%”。

本轮已经满足以上停止标准。后续如果引入新需求、新依赖、新架构或真实压测数据，必须重新开启本节的漏洞复盘循环。

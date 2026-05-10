# OpsPilot V1.0 Release Evidence

> Language: 简体中文（当前） | [English](./release-evidence.en.md)

本文档记录 V1.0 产品化验收过程中的命令、结果、环境和风险。当前状态为正式 `v1.0.0` 发布验收记录。

## V1.0 更迭记录

| 日期 | 版本/提交 | 更迭内容 | 同步说明 |
| 2026-05-10 | V1.0 加固中 / 待提交 | 修复 Frontend CI `npm ci` 锁文件不一致导致的安装失败风险。 | `web/package-lock.json` 补齐 `@emnapi/core`/`@emnapi/runtime`/`@napi-rs/wasm-runtime` 传递依赖并统一 npmjs registry URL；`web/.npmrc` 固定官方 registry 且开启 `replace-registry-host=always`；CI 前端安装命令显式 `--registry=https://registry.npmjs.org`；本地以 `npm@11.11.0` 复验 `npm ci --include=optional` 通过。 |
| 2026-05-10 | V1.0 加固中 / 待提交 | Incident 生命周期闭环、告警噪声治理最小版、CI 漏洞门禁与运维演练脚本。 | 新增 Incident lifecycle/merge/close API+UI+OpenAPI+测试；新增 alerts noisy rules / suppression dry-run / routing dry-run / noise trends API+UI+OpenAPI+测试；Playwright smoke 补 incidents 与 trace-center；CI 增加 govulncheck 与 npm audit；新增 `scripts/db-backup.ps1` 与 V1.1 运维加固演练手册。 |
| 2026-05-09 | `v1.0.0` 发布收口 | 根版本号切换为 `1.0.0`，远端 CI 已确认通过。 | GitHub Actions 已在 `master` 与 `v1.0.0-beta.1` tag 上完成绿色验证；V1.0 发布阻塞项收敛为 V1.1 加固计划。 |
| 2026-05-09 | V1.0 开发中 / 待提交 | Docker/Compose 发布验收与全栈 smoke 已在本机闭环。 | 移除 Dockerfile frontend 远程依赖，Compose 改为参数化发布端口并去掉固定 `container_name`，补齐 API/Agent 时区配置以修正 wait/schedule 的跨时区偏差，扩展 `scripts/e2e-smoke.ps1`，并在 CI 中加入 full-stack E2E 与 notification config 落盘加密校验。 |
| 2026-05-09 | V1.0 寮€鍙戜腑 / 寰呮彁浜?| 修复 CI 中 schedule smoke 的时区错配。 | E2E smoke 脚本改为对 schedule workflow 校验使用与 runner 本地时区无关的每分钟 cron，并在首个 fired trigger 后立即 disable smoke schedule，避免 GitHub Actions runner 时区与 `Asia/Shanghai` 不一致时出现假失败。 |
| 2026-05-09 | V1.0 开发中 / 待提交 | 清理 GitHub Actions 剩余的运行时弃用风险。 | CI 已升级到 `actions/checkout@v5`、`actions/setup-node@v6`、`actions/setup-go@v6`，让 workflow 直接运行在原生 Node 24 action runtime 上。 |
| 2026-05-09 | V1.0 开发中 / 待提交 | 补齐严格 V1.0 范围下剩余的 matcher 和文档缺口。 | 后端/前端/OpenAPI/E2E smoke 已补齐 `payload_exists`、`payload_not_equals`、`payload_regex` 和 JSONPath 数组通配 matcher，同时新增 V1.0 部署手册、验收手册、UI 规范。 |
| 2026-05-09 | V1.0 in progress / pending | Squashed the active V1.0 SQL migration chain. | Replaced root 000005-000015 with 000005_v10_productization_bundle, archived the original incremental files, validated MySQL apply/rollback and schema equivalence against the legacy chain, and aligned VERSION to 1.0.0. |
| 2026-05-09 | V1.0 开发中 / 待提交 | 前端 i18n 收尾一轮。 | 清理 Workflow/Audit/Agent/Webhook 剩余硬编码文案，补齐默认示例、导出失败提示、actor type/result 展示与 UI 操作原因的 zh/en 词条；同步 README、CHANGELOG、全局 summary、V1.0 设计书；验证前端 build 与 `git diff --check`。 |
| 2026-05-08 | V1.0 开发中 / 待提交 | Alert grouping 运营视图。 | 同步 README、CHANGELOG、OpenAPI、UML、全局 summary、V1.0 设计书；验证 Go test/vet、前端 build、迁移配对、OpenAPI/router、git diff check。 |
| 2026-05-08 | V1.0 开发中 / 待提交 | Agent Host Group 批量操作与 group scope 维护窗口。 | 同步 README、CHANGELOG、OpenAPI、UML、全局 summary、V1.0 设计书；验证 Go test/vet、前端 build、迁移配对、OpenAPI/router。 |
| 2026-05-08 | V1.0 开发中 / 待提交 | Alert suppression/routing 增加 Host Group 匹配能力。 | 同步 OpenAPI、UML、全局 summary、V1.0 设计书；验证 Go test/vet、前端 build、迁移配对、OpenAPI/router。 |
| 2026-05-08 | V1.0 开发中 / 待提交 | Metrics rollup/retention 后台 worker，新增可配置维护周期。 | 同步 README、`.env.example`、Compose、UML、全局 summary、V1.0 设计书；验证 Go test/vet、迁移配对、OpenAPI/router。 |
| 2026-05-08 | `3870522` | Maintenance window 接入 alert 通知抑制与 schedule 触发跳过。 | 同步 UML 与本 Release Evidence；验证 Go test/vet、迁移配对、OpenAPI/router。 |
| 2026-05-08 | `412fd6d` | Agent/Host tags、Host Group API/UI。 | 同步 OpenAPI、UML、本 Release Evidence；验证 Go test/vet、前端 build、OpenAPI/router。 |
| 2026-05-08 | `b443390` | Agent diagnostics、version inventory、maintenance windows API/UI。 | 同步迁移、OpenAPI、UML、本 Release Evidence；验证迁移配对、Go test/vet、前端 build。 |
| 2026-05-08 | `ebe1dec` | Alert suppression rules 与 routing policies。 | 同步 OpenAPI、UML、本 Release Evidence；验证后端/前端检查。 |
| 2026-05-08 | `ff792ad` | Metrics rollup、retention、saved dashboard、趋势粒度。 | 同步 OpenAPI、UML、本 Release Evidence；验证后端/前端检查。 |
| 2026-05-08 | `82de7fb`、`e1cd3d9` | Notification 模板、channel-specific template、批量 retry。 | 同步 OpenAPI、本 Release Evidence；验证后端/前端检查。 |
| 2026-05-08 | `422b5e5` | Workflow 节点 retry、timeout、失败策略。 | 同步 UML、本 Release Evidence；验证后端/前端检查。 |
| 2026-05-08 | `3a05248` | V1.0 Workflow、Webhook、Audit、Secret 治理和发布验收骨架。 | 建立 OpenAPI/router diff、E2E smoke、Release Evidence、UML 更新。 |

## 2026-05-09 SQL Migration Squash Validation

Commands executed:

```powershell
cd D:\+\1108026_rust_go\OpsPilot
powershell -File .\scripts\migration-check.ps1 `
  -HostName 127.0.0.1 -Port 3306 `
  -Database opspilot_migcheck_squash_1778261001 `
  -User root -Password 123456 `
  -MysqlExe "C:\Program Files\MySQL\MySQL Server 8.0\bin\mysql.exe"
```

Results:

| Area | Result |
| --- | --- |
| Active migration pair check | Passed, 5 root up/down migration pairs. |
| Squashed V1.0 apply | Passed, 000005_v10_productization_bundle.up.sql applied after 000001-000004. |
| Squashed V1.0 rollback | Passed, 000005_v10_productization_bundle.down.sql restored the schema to the pre-V1.0 state. |
| Legacy-chain equivalence | Passed, normalized schema dumps and table row-count signatures match the archived 000005-000015 chain. |

Notes:

- The active root migration path is now optimized for new environment bootstrap.
- Historical incremental V1.0 migrations remain available in server/migrations/archive/v1.0-incremental/.

## 2026-05-09 发布验收快照

环境：

- OS: Windows / PowerShell
- Node.js: 本机 Node 24 工具链
- Docker: 本机可用
- 工作区：`D:\+\1108026_rust_go\OpsPilot`

已执行命令：

`````powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
go vet ./...

cd D:\+\1108026_rust_go\OpsPilot\web
npm install --no-audit --no-fund
npm run build

cd D:\+\1108026_rust_go\OpsPilot
powershell -File .\scripts\migration-check.ps1 -SkipExecution
powershell -File .\scripts\openapi-router-check.ps1
docker compose -f deploy/docker-compose.yml --profile full config
docker build -f server/Dockerfile --build-arg APP_VERSION="$(Get-Content VERSION)" .
docker build -f web/Dockerfile .

$env:MYSQL_PUBLISHED_PORT="33306"
$env:REDIS_PUBLISHED_PORT="36379"
$env:API_PUBLISHED_PORT="18080"
$env:WEB_PUBLISHED_PORT="15173"
$env:SCHEDULE_SCAN_INTERVAL_SECONDS="2"
$env:ALERT_SCAN_INTERVAL_SECONDS="2"
$env:NOTIFICATION_DISPATCH_INTERVAL_SECONDS="2"
$env:METRIC_ROLLUP_INTERVAL_SECONDS="3600"
$env:METRIC_RETENTION_INTERVAL_SECONDS="3600"
docker compose -p opspilot-local-e2e -f deploy/docker-compose.yml --profile full up --build -d
powershell -File .\scripts\e2e-smoke.ps1 -ApiBaseUrl http://127.0.0.1:18080 -ScheduleTimeoutSeconds 180
docker compose -p opspilot-local-e2e -f deploy/docker-compose.yml exec -T mysql mysql -uopspilot -popspilot opspilot -Nse "SELECT config FROM notification_channels WHERE config IS NOT NULL ORDER BY id DESC LIMIT 1;"
```

结果：

| 项 | 结果 |
| --- | --- |
| 后端测试 | 通过，`go test ./...` 全绿。 |
| 后端静态检查 | 通过，`go vet ./...` 无输出。 |
| 前端构建 | 通过，`tsc --noEmit && vite build` 成功。 |
| 迁移配对 | 通过，当前根迁移 5 对 up/down。 |
| OpenAPI/router diff | 通过，148 个 router routes 已记录到 OpenAPI。 |
| Docker compose config | 通过。 |
| Docker 镜像构建 | 通过，`server/Dockerfile` 与 `web/Dockerfile` 均可构建。 |
| Compose 全栈 smoke | 通过，MySQL、Redis、migrate、API、Web、Agent 同时启动并完成验收。 |
| E2E smoke | 通过，覆盖 manual/schedule/webhook workflow、run cancel、整 run retry、node retry、matcher simulator、webhook replay、notification masking、audit export/retention。 |
| Secret 落盘加密 | 通过，最新 `notification_channels.config` 为 `"_encrypted"` JSON，不包含明文 smoke secret。 |

备注：

- API/Web Dockerfile 已移除 syntax directive，Docker build 不再依赖额外拉取 `docker/dockerfile:1` frontend 镜像。
- Compose 已改为参数化发布端口、移除固定 `container_name`，并为 API/Agent 显式设置 `TZ`，避免数据库与应用时区不一致时 wait/schedule 发生时间漂移。
- `.github/workflows/ci.yml` 现会拉起 full Compose profile、执行 `scripts/e2e-smoke.ps1`、校验 notification config 落盘加密，并在结束后统一清理栈。

## 当前 V1.0 已具备

- Workflow 创建、更新、发布、禁用、复制、版本列表、手动输入运行、取消、整 run 重试、节点级重试、审批节点处理、节点 `timeoutSeconds` 和 `stop_workflow` / `skip_downstream` / `continue` 失败策略。
- Workflow 权限已收敛到 `workflow:read`、`workflow:manage`、`workflow:execute`，旧的 `workflow:write` / `workflow:cancel` 作为兼容别名保留。
- Schedule/Webhook 触发 Workflow。
- Workflow task、condition、approval、notification、wait、webhook-call 基础节点执行。
- Notification/Webhook secret 加密存储与 rotation 基础能力；Notification template、channel-specific template 渲染，以及 delivery 单条/批量失败重试。
- Audit export、retention run 和 advanced search 的部分过滤能力。
- Webhook matcher simulator 和 event replay。
- Incident 投影模型、列表、详情和 timeline。
- Alert grouping 运营视图，支持按 rule、host group、severity、fingerprint 聚合。
- Alerts suppression rules、routing policies 已具备 API/UI，并在告警触发时支持按 rule/host/host group/severity 抑制或定向投递到指定通知渠道。
- Metrics 5m/1h rollup、明细/rollup retention dry-run/执行入口、后台 rollup/retention worker、saved dashboard API/UI、趋势接口 `auto/raw/5m/1h` 粒度选择。
- Agent diagnostics 快照、version inventory、tags/groups 和 maintenance window 已具备 API/UI；Host Group 支持批量禁用 Agent 和按组查看 diagnostics；Heartbeat 会写入最新诊断快照；维护窗口支持 all/agent/host/group 范围并会抑制告警通知、跳过维护期调度触发。
- 前端共享 DataTable、FilterToolbar、Confirm、Toast、Timeline、JsonViewer 等基础组件。
- CI 已覆盖 migration pair、OpenAPI/router diff、Go test/vet、npm build、Docker build、compose config，以及带落盘加密断言的 full-stack Compose smoke。

## V1.0.0 发布收口状态

- 远端 GitHub Actions 已在 `master` 与 `v1.0.0-beta.1` tag 上通过，V1.0 的 CI 发布门槛已满足。
- 根版本号已从 `1.0.0-dev` 收口为 `1.0.0`。
- 原始 V1.0 增量迁移链保留在 `server/migrations/archive/v1.0-incremental/`，当前根迁移路径使用 `000005_v10_productization_bundle`。
- 最终 release report 可继续补充关键页面桌面/窄屏截图证据，优先覆盖 Dashboard、Workflow、Notifications、Audit；这属于发布证据增强，不再作为 V1.0 阻塞项。
- Workflow retry/timeout、Metrics rollup/retention、Alert grouping 等路径已经具备可运行基础能力，更深的数据库集成覆盖进入 V1.1 加固计划。

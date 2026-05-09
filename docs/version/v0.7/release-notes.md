# OpsPilot V0.7 发布说明

> Language: 简体中文（当前） | [English](./release-notes.en.md)

生成日期：2026-05-08。

## 版本定位

V0.7 是稳定化版本。目标不是继续增加大型产品域，而是让已有 T1-T8 能力达到可 CI 验证、可演示、可部署和可按生产近似环境验收的状态。

当前版本标记：`0.7.0-dev`。

## 本轮已完成

### Release Engineering

- GitHub Actions CI 覆盖后端测试、`go vet`、迁移配对检查、前端构建和 Docker build。
- `VERSION` 与 `/api/v1/version` 暴露构建身份。
- API/Web Dockerfile 和 Compose `full` profile 支持全栈启动。
- OpenAPI 已同步 V0.7 核心 API 面。
- E2E demo、安全权限矩阵、安全检查清单和 changelog 已作为发布材料纳入仓库。

### Security Baseline

- 生产环境默认关闭公开注册。
- 生产环境 JWT secret 会被校验。
- CORS 使用明确 allowlist。
- 用户 JWT、Agent token、Webhook token 的权限边界已有测试覆盖。
- Webhook trigger token 与 signing secret 已拆分。
- 日志和审计 payload 会遮盖 token/password/secret/authorization 类敏感值。

### Webhooks

- 已实现 timestamp、nonce 和 delivery-id 防重放。
- 触发失败会落库并记录 failure reason。
- Rule matcher 支持 header、eventType、JSON field、ref 和 branch 上的 AND 条件。
- Webhook event 查询支持 source、status、deliveryId 和 received time range。
- Source/rule 生命周期操作和事件调试已在 UI 中可用。

### Schedules

- Cron next preview 可在创建 schedule 前预览未来触发时间。
- Misfire policy 已接入创建和扫描逻辑，支持 `skip`、`fire_once`、`fire_all`。
- Trigger history API/UI 可查看最近触发记录、task_run 和失败原因。
- Cron parser 测试覆盖 step、range、list、Sunday、非法表达式和 timezone。

### Metrics And Alerts

- Agent 上报真实 OS 指标，包括 CPU、内存、磁盘和网络。
- Metrics trend API 与前端趋势图已可用。
- Alerts 支持 ack、silence、unsilence、resolve、cooldown、恢复通知、history 和 event detail。
- Alert duration window 会要求持续超阈值后再触发。

### Notifications

- SMTP Email sending 已通过环境变量实现。
- Delivery 查询、手动重试和 channel test send 已实现。
- Channel 列表通过 `targetSummary` 做目标遮盖。
- Outbound webhook notification 已添加 `X-OpsPilot-*` 签名头。

### Frontend Experience

- Metrics、Alerts、Notifications 页面补齐了更多状态、筛选和操作。
- Audit Logs 页面已接入 `audit.read` 权限，可检索审计事件并查看脱敏后的 before/after/metadata。
- UI language switching 继续通过现有 i18n 层支持；Agent、Host、Webhook、Metric、SMTP、HMAC 等产品和技术术语不翻译。
- 视觉样式继续向统一的运维控制台主题收敛。

## 打 Tag 前必跑验证

执行：

```powershell
cd server
go test ./...
go vet ./...

cd ..\web
npm run build

cd ..
powershell -File .\scripts\migration-check.ps1 -SkipExecution
```

Docker 可用时还需要执行：

```powershell
docker compose -f deploy/docker-compose.yml --profile full config
docker build -f server/Dockerfile .
docker build -f web/Dockerfile .
```

## 已知缺口

- Notification channel config 在 API/log/audit 边界已遮盖，但尚未加密存储。
- Docker 镜像 build 需要在可用 Docker 环境中执行最终验收。

## 建议后续实现顺序

1. 在 Docker 可用机器上补充 Docker 验证并记录结果。
2. 增强 Audit Logs 的导出、保留周期和高级检索能力。
3. 为 Notification channel config 增加加密存储。
4. 验证保持通过后，从干净 release branch 准备 `v0.7.0` tag。

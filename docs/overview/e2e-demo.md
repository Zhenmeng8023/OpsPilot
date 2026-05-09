# OpsPilot E2E Demo

本文档用于 V0.7 手工验收：从启动基础设施到登录、Agent 接入、脚本执行、日志查看、Cron、Webhook、Metrics、Alert、Notification 跑通一条可复现链路。

## 1. 前置要求

- Go 1.26.2
- Node.js 24.x
- npm 11.x
- MySQL 8.0.39
- Redis 7.x
- Windows PowerShell

默认账号来自 seed：

```text
username: admin
password: Admin@123456
```

## 2. 启动方式 A：本地开发进程

启动 MySQL 和 Redis：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
docker compose -f .\deploy\docker-compose.yml up -d
```

执行迁移：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1
```

启动 API：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
Copy-Item .\.env.example .\.env -ErrorAction SilentlyContinue
go run .\cmd\api
```

启动 Web：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\web
Copy-Item .\.env.example .\.env -ErrorAction SilentlyContinue
npm install
npm run dev
```

启动 Agent：

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
$env:AGENT_API_BASE_URL='http://localhost:8080'
$env:AGENT_BOOTSTRAP_SECRET='change-me-agent-bootstrap-secret'
$env:AGENT_WORK_DIR='.tmp/agent-work'
$env:AGENT_TOKEN_FILE='.tmp/agent-token'
$env:AGENT_POLL_INTERVAL_SECONDS='5'
$env:AGENT_HEARTBEAT_INTERVAL_SECONDS='30'
go run .\cmd\agent
```

打开：

```text
http://localhost:5173
```

## 3. 启动方式 B：Compose 全栈

全栈 profile 会额外启动 migrate、API、Web、Agent：

```powershell
cd D:\+\1108026_rust_go\OpsPilot
Copy-Item .\deploy\.env.example .\deploy\.env -ErrorAction SilentlyContinue
docker compose --env-file .\deploy\.env -f .\deploy\docker-compose.yml --profile full up --build
```

打开：

```text
http://localhost:5173
```

当前本地环境如果没有 Docker CLI，可跳过方式 B；CI 会执行 Docker build。

## 4. 基础健康检查

```powershell
Invoke-RestMethod http://localhost:8080/health
Invoke-RestMethod http://localhost:8080/api/v1/version
```

验收点：

- `/health` 返回 `status=ok`。
- `/api/v1/version` 返回 `version`、`goVersion`、`env`。
- Dashboard 能展示 API health 和版本信息。

## 5. 登录和 Agent 接入

1. 使用 `admin / Admin@123456` 登录。
2. 进入 Agents 页面。
3. 确认至少 1 个 Agent 为 `online`。
4. 停止 Agent 进程，等待离线扫描周期后点击 Scan offline，确认 Agent 或 Host 状态变为 offline。
5. 重新启动 Agent，确认心跳恢复。

## 6. 脚本任务和实时日志

创建脚本：

```powershell
Write-Output "OpsPilot demo start"
Get-Date
Write-Output "secret token=should_be_redacted"
Start-Sleep -Seconds 3
Write-Output "OpsPilot demo done"
```

操作步骤：

1. 进入 Scripts 页面，创建 PowerShell 脚本。
2. 如脚本启用了 approval required，提交审批并在审批区 approve。
3. 进入 Tasks 页面创建任务，选择该脚本和在线 Agent。
4. 打开任务详情，确认 target 状态从 queued/running 进入 success。
5. 打开日志终端，确认 stdout/system 日志实时出现。
6. 检查敏感词是否被脱敏。

## 7. Cron 调度

1. 进入 Schedules 页面。
2. 选择刚才的任务定义，Cron 使用：

```text
*/5 * * * *
```

3. 等待下一个触发窗口。
4. 回到 Tasks 页面，确认由 schedule 触发的新 task run 出现。
5. 暂停、恢复、禁用该 schedule，确认状态变更成功。

## 8. Webhook 触发

创建 webhook source 和 rule：

1. 进入 Webhooks 页面。
2. 创建 source，保存一次性展示的 token。
3. 创建 rule，绑定到已有任务定义，event type 可填 `push`。

发送模拟请求。签名使用 source token 作为当前版本 HMAC key：

```powershell
$token = '<source-token>'
$body = '{"event":"demo","ref":"refs/heads/main"}'
$key = [Text.Encoding]::UTF8.GetBytes($token)
$bytes = [Text.Encoding]::UTF8.GetBytes($body)
$hmac = [System.Security.Cryptography.HMACSHA256]::new($key)
$signature = 'sha256=' + (($hmac.ComputeHash($bytes) | ForEach-Object { $_.ToString('x2') }) -join '')

Invoke-RestMethod `
  -Method Post `
  -Uri "http://localhost:8080/api/v1/webhooks/trigger/$token" `
  -ContentType 'application/json' `
  -Headers @{
    'X-OpsPilot-Signature' = $signature
    'X-Delivery-Id' = [guid]::NewGuid().ToString()
    'X-Event-Type' = 'push'
  } `
  -Body $body
```

验收点：

- 返回包含 triggered run。
- Tasks 页面出现 webhook 触发的 task run。
- 使用错误签名再次请求，应返回 401。
- 使用相同 `X-Delivery-Id` 重放，应返回 409。

## 9. Metrics、Alerts、Notifications

1. 保持 Agent 运行，等待一次 metrics 上传。
2. 进入 Metrics 页面，选择 `agent.runtime.goroutines` 或 `agent.runtime.alloc_bytes`。
3. 创建一个容易触发的告警规则，例如：

```text
metric: agent.runtime.goroutines
operator: gt
threshold: 0
severity: warning
duration: 60
```

4. 等待 alert scanner 周期。
5. 确认 firing alert 出现。
6. 进入 Notifications 页面，确认生成站内通知和 delivery。
7. 点击 Resolve，确认告警可手动恢复。

扩展验收点：

- Email/SMTP sender 可通过 `NOTIFICATION_SMTP_*` 环境变量配置，授权码只放入本地 `.env`，不要提交到仓库。
- Alert ack/silence/cooldown/恢复通知已接入，可在 Metrics 页面继续验证状态流转。
- Agent OS CPU/内存/磁盘/网络指标已采集，可在 Metrics 页面查看趋势。
- Webhook timestamp/nonce、失败事件、matcher 结果和时间范围检索已接入 Webhooks 页面。

## 10. 必跑验证命令

```powershell
cd D:\+\1108026_rust_go\OpsPilot\server
go test ./...
go vet ./...

cd D:\+\1108026_rust_go\OpsPilot\web
npm run build

cd D:\+\1108026_rust_go\OpsPilot
.\scripts\migration-check.ps1 -SkipExecution
```

## 11. 常见问题

### 登录失败

确认迁移和 seed 已执行，数据库中存在 `admin` 用户。必要时重建本地 MySQL volume 后重新执行迁移。

### Agent 注册失败

确认 `AGENT_BOOTSTRAP_SECRET` 与 API 的 `AGENT_BOOTSTRAP_SECRET` 一致，并且 `AGENT_REGISTRATION_ENABLED=true`。

### 前端无法访问 API

本地 Vite 模式下确认 API 监听 `localhost:8080`。Compose 全栈模式下确认 Nginx 代理 `/api/` 到 `api:8080`。

### Webhook 签名失败

V0.7 已拆分 trigger token 与 signing secret。触发 URL 中的 token 只用于定位 source，HMAC key 使用创建 source 时一次性展示的 signing secret。

### 没有通知

确认 channel 状态为 active，dispatcher 正在运行。Email channel 还需要在 API 环境中配置 `NOTIFICATION_SMTP_HOST`、`NOTIFICATION_SMTP_PORT`、`NOTIFICATION_SMTP_USERNAME`、`NOTIFICATION_SMTP_PASSWORD`、`NOTIFICATION_SMTP_FROM` 等变量。

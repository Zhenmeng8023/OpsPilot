# Changelog

> Language: 简体中文（当前） | [English](./CHANGELOG.en.md)

## Unreleased

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

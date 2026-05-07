# OpsPilot V0.7 安全权限矩阵

> Language: 简体中文（当前） | [English](./security-permission-matrix.en.md)

本文档用于 V0.7 发布验收，记录当前 RBAC 权限面、内置角色和前后端路由门禁。

## 权限编码

| 模块 | 规范权限 | 兼容别名 | 主要能力 |
| --- | --- | --- | --- |
| workspace | `workspace.read` | - | 查看 Workspace 与 Dashboard 元数据。 |
| user | `user.read` | - | 查看用户与成员关系。 |
| user | `user.write` | - | 创建和更新用户。 |
| role | `role.read` | - | 查看角色与权限目录。 |
| role | `role.write` | - | 管理角色权限分配。 |
| agent | `agent:read` | `agent.read` | 查看 Agent 与 Host。 |
| agent | `agent:write` | `agent.disable` | 禁用 Agent、撤销 token。 |
| host | `host:read` | - | 查看 Host 清单。 |
| host | `host:write` | - | 管理 Host 元数据。 |
| script | `script:read` | `script.read` | 查看脚本与版本。 |
| script | `script:write` | `script.write` | 创建和更新脚本。 |
| script | `script:approve` | `script.approve` | 审批高风险脚本版本。 |
| task | `task:read` | `task.read` | 查看 task run 与 target。 |
| task | `task:write` | - | 管理任务定义。 |
| task | `task:execute` | `task.run` | 创建和执行任务。 |
| task | `task:cancel` | `task.cancel` | 取消排队或运行中的任务。 |
| task | `task:log:read` | `log.read` | 查看和流式消费任务日志。 |
| schedule | `schedule:read` | - | 查看调度任务。 |
| schedule | `schedule:write` | `schedule.write` | 创建和管理调度任务。 |
| metric | `metric:read` | `metric.read` | 查看 Host metrics、趋势和告警。 |
| alert | `alert:read` | - | 查看告警规则、告警、事件和历史。 |
| alert | `alert:write` | `alert.write` | 管理告警规则和告警状态。 |
| webhook | `webhook:read` | - | 查看 Webhook source、rule 和 event。 |
| webhook | `webhook:manage` | `webhook.manage` | 管理 Webhook source 和 trigger rule。 |
| notification | `notification:read` | - | 查看通知渠道、通知消息和 delivery。 |
| notification | `notification:write` | `notification.write` | 管理通知渠道、重试和测试 delivery。 |
| audit | `audit.read` | - | 查看审计日志。 |

## 内置角色

| 角色 | 权限 |
| --- | --- |
| `admin` | 所有 seed 权限。 |
| `member` | `workspace.read`、`agent:read`、`host:read`、`script:read`、`task:read`、`task:log:read`、`metric:read`。 |

## 路由门禁摘要

| 页面/接口面 | 读权限 | 写入/执行权限 |
| --- | --- | --- |
| Dashboard | `workspace.read` | - |
| Users | `user.read` | `user.write` |
| Roles | `role.read` | `role.write` |
| Agents and Hosts | `agent:read`、`host:read` | `agent:write`、`host:write` |
| Scripts | `script:read` | `script:write`、`script:approve` |
| Tasks | `task:read`、`task:log:read` | `task:execute`、`task:cancel` |
| Schedules | `schedule:read` | `schedule:write` |
| Webhooks | `webhook:read` | `webhook:manage` |
| Metrics and Alerts | `metric:read`、`alert:read` | `alert:write` |
| Notifications | `notification:read` | `notification:write` |
| Audit Logs | `audit.read` | - |

## 兼容说明

- 后端权限检查通过 `permissionMatchCodes` 同时接受规范权限和历史别名。
- 角色分配时会尽量把别名归一到规范权限。
- 前端路由守卫保留对历史 dot-style 权限的兼容。
- V0.7 新代码优先使用 colon-style 权限；`workspace.read`、`user.read`、`role.read` 等历史 workspace/admin 权限继续保留 dot-style。

# OpsPilot V1.0 UI 规范

> Language: 简体中文（当前） | [English](./ui-guidelines.en.md)

本文档定义 V1.0 页面在实现和验收时必须满足的最低 UI 标准。

## 1. 布局与响应式

- 关键页面在桌面和窄屏下都不能出现不可控横向溢出
- 表格需要通过换行、截断或次级文本堆叠保持可读
- 筛选区和操作区在收窄后不能隐藏关键操作
- JSON payload、timeline、audit/event 详情必须保持可滚动、可阅读

## 2. 国际化

- 用户可见文案必须支持 zh/en
- `Agent`、`Host`、`Webhook`、`Metric`、`SMTP`、`HMAC`、`JSONPath` 这类技术词可以保留英文
- 默认示例、校验提示、动作原因、空状态也必须完成国际化

## 3. 共享组件

V1.0 页面默认复用以下共享组件：

- `DataTable`
- `FilterToolbar`
- `JsonViewer`
- `PaginationBar`
- `Timeline`
- `ConfirmDialog`

## 4. 安全展示规则

- secret 在创建或轮换后，除 API 明确定义的一次性展示场景外，不得再次明文回显
- API / audit 错误详情如果可能包含 token、password、signing secret，必须保持脱敏
- Webhook / Audit / Notification 页面优先展示脱敏摘要，而不是原始敏感 payload

## 5. Webhook 页面规则

- 规则编辑器必须暴露完整 V1.0 matcher 集，包括 `payload_exists`、`payload_not_equals`、`payload_regex` 和数组通配路径
- matcher simulator 必须展示逐条件结果、actual 值和失败原因
- event 详情必须展示 headers、payload、安全状态以及关联 task/workflow run

## 6. 验收期望

页面只要引入新的 workflow、webhook、audit、notification、alert 行为，就必须至少检查：

- zh/en 文案
- 响应式安全布局
- 权限控制下的动作展示
- 敏感数据的安全渲染

# OpsPilot 文档目录规范

本文定义 `docs/` 目录的长期文件划分规则，避免不同版本、不同功能、不同用途的文档混放。

## 1. 总原则

1. 根目录只保留文档入口和跨版本总览，不继续堆具体版本文件。
2. 版本相关文档必须进入 `docs/version/<version>/`。
3. 发布验收、部署、UI、设计、迁移说明如果只属于某个版本，就放到该版本目录。
4. 跨版本长期有效的架构、安全、UML、开发流程可以放到独立主题目录。
5. 中英文双语文件放在同一目录，中文默认文件为 `*.md`，英文为 `*.en.md`。
6. 新文档必须能从 `docs/README.md` 或对应版本 `README.md` 找到入口。

## 2. 目标目录结构

建议最终收敛为：

```text
docs/
  README.md
  overview/
    project-summary.md
    e2e-demo.md
  architecture/
    uml/
      core-uml.md
  security/
    permission-matrix.md
    permission-matrix.en.md
  version/
    product-development-roadmap.md
    v0/
      original-technical-plan.docx
      database-design-mysql.md
    v0.7/
      README.md
      next-version-plan.md
      release-notes.md
      release-notes.en.md
      security-release-checklist.md
      security-release-checklist.en.md
    v1.0/
      README.md
      product-development-design.md
      product-development-design.en.md
      release-evidence.md
      release-evidence.en.md
      deployment-guide.md
      deployment-guide.en.md
      acceptance-guide.md
      acceptance-guide.en.md
      ui-guidelines.md
      ui-guidelines.en.md
    v1.1/
      README.md
      product-development-design.md
```

## 3. 目录职责

| 目录 | 放什么 | 不放什么 |
| --- | --- | --- |
| `docs/` | 文档入口、目录规范 | 某个版本的验收/部署/UI 细节 |
| `docs/overview/` | 跨版本项目总览、演示流程 | 单版本 release evidence |
| `docs/architecture/` | 跨版本架构、UML、数据库基线 | 某个版本的发布说明 |
| `docs/security/` | 跨版本权限矩阵、安全基线 | 某版本临时安全 checklist |
| `docs/version/` | 长期路线图和版本目录 | 零散功能文档 |
| `docs/version/<version>/` | 该版本设计、发布、验收、部署、UI、变更说明 | 其他版本或跨版本材料 |

## 4. 当前文件归类

当前仓库已有文件已按下面方式归类。后续新增文档应继续遵守此表。

| 文件 | 位置 | 说明 |
| --- | --- | --- |
| 项目总览与演示 | `docs/overview/` | 跨版本说明 |
| 安全权限矩阵 | `docs/security/permission-matrix*.md` | 跨版本安全基线 |
| 通用 UML | `docs/architecture/uml/core-uml.md` | 跨版本架构总览 |
| V0 原始方案 | `docs/version/v0/` | 历史基线 |
| V0.7 文档 | `docs/version/v0.7/` | 版本归档 |
| V1.0 文档 | `docs/version/v1.0/` | 版本设计、发布、部署、验收、UI、UML 快照 |
| V1.1 文档 | `docs/version/v1.1/` | 下一版本设计 |
| 中长期路线图 | `docs/version/product-development-roadmap.md` | 跨版本路线图 |

## 5. 命名规则

- 版本目录：`v1.0`、`v1.1`、`v2.0`。
- 版本目录内文件不再重复版本号，例如使用 `release-evidence.md`，不使用 `v1.0-release-notes.md`。
- 跨版本文件使用稳定主题名，例如 `permission-matrix.md`、`core-uml.md`。
- 英文版本统一使用 `.en.md`。
- 不再新增空格、中文标点或难以脚本处理的文件名；历史文件可以保留，迁移时再重命名。

## 6. 版本目录标准内容

每个正式版本目录建议包含：

```text
docs/version/vX.Y/
  README.md
  product-development-design.md
  release-evidence.md
  deployment-guide.md
  acceptance-guide.md
  ui-guidelines.md
```

不是每个版本都必须有全部文件。小版本至少需要：

- `README.md`
- `product-development-design.md`
- `release-evidence.md` 或在 CHANGELOG 中明确说明无需独立验收记录

## 7. 迁移和调整检查

文档已经按版本目录完成归档。后续如果继续移动、拆分或重命名文档，每一步都必须执行：

- `rg` 检查旧路径引用。
- 修复 README、CHANGELOG、版本设计书中的链接。
- `git diff --check`。

## 8. 新文档准入规则

新增文档前先判断：

- 是否只属于一个版本：放 `docs/version/<version>/`。
- 是否跨版本长期有效：放 `docs/overview/`、`docs/architecture/` 或 `docs/security/`。
- 是否是路线图：放 `docs/version/`。
- 是否是 API 契约：优先放 `server/docs/openapi/`，`docs/` 只放说明。
- 是否是迁移 SQL：放 `server/migrations/`，`docs/` 只放迁移设计或验收证据。

如果找不到合适目录，先更新本文档，不要把文件直接丢到 `docs/` 根目录。

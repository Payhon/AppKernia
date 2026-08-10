---
name: project-docs-governance
description: 为编码智能体项目初始化或补齐文档优先的治理体系。用于从 0 创建项目、为已有代码但缺少文档规范的项目建立 docs 分层目录、功能开发文档、质量与运维记录、项目进度看板，并以非破坏方式将新增/更新/使用规则写入 AGENTS.md；适用于 Codex、GLM5、Claude Code 等参与的软件开发流程。
---

# Project Docs Governance

在当前项目建立“先文档、后开发、过程中留痕、完成后回写”的可执行约束。默认只新增缺失文件，不覆盖既有文档。

## 执行流程

1. 将当前工作目录视为项目根目录；用户给出路径时使用指定路径。
2. 检查项目根目录、现有 `docs/`、`AGENTS.md` 和代码清单，确认没有把子目录误当项目根目录。
3. 选择模式：
   - `new`：从 0 开始或尚无业务代码的项目。
   - `existing`：已有代码，需要补建文档治理并保留原有文档。
   - `auto`：由脚本根据代码、构建清单和版本库状态判断；无法确定时使用 `existing`，因为它更保守。
4. 先预览，再创建：

   ```bash
   python3 .agents/skills/project-docs-governance/scripts/bootstrap_project_docs.py \
     --project-root "$PWD" --mode auto --dry-run

   python3 .agents/skills/project-docs-governance/scripts/bootstrap_project_docs.py \
     --project-root "$PWD" --mode auto
   ```

5. 运行只读验收：

   ```bash
   python3 .agents/skills/project-docs-governance/scripts/bootstrap_project_docs.py \
     --project-root "$PWD" --check
   ```

6. 阅读脚本报告，人工核对保留、创建和更新的文件。不得把“脚本运行成功”冒充项目内容已经完成梳理。

如需把完整任务交给其他编码智能体，不要临时重写要求：新项目读取 `references/prompt-new-project.md`，已有项目读取 `references/prompt-existing-project.md`。

## 安全与更新规则

- 默认保留所有已存在文件；只创建缺失文件。
- 仅更新 `AGENTS.md` 中 `DOCS_GOVERNANCE` 受管标记之间的内容，保留文件中其他规则。
- 只有用户明确要求重建生成文件时才使用 `--force`。使用前先检查目标文件差异。
- 不删除、移动或重命名已有文档，不修改业务代码。
- 已有项目的 `current-state-baseline.md` 是目录级初始盘点，不是完整架构审计；必须在后续开发中根据真实代码继续校正。
- 当前项目已有文档时，将其登记到索引或现状基线，不得为迎合新层级而强行迁移。

## 两种模式的交付差异

### 新项目

- 创建项目章程、初始路线图、架构占位和质量/运维基线。
- 只建立“文档治理初始化”工作项，不虚构业务需求或技术选型。
- 在看板 Backlog 中保留首个业务功能的待定义入口。
- 需要提供给其他编码智能体的完整提示词时，读取并原样适配 `references/prompt-new-project.md`。

### 已有项目

- 保留代码、目录和既有文档。
- 基于清单生成当前状态基线，明确“自动发现”和“待人工确认”的边界。
- 将治理初始化作为已完成工作项记录；将项目文档盘点放入 Backlog。
- 需要提供给其他编码智能体的完整提示词时，读取并原样适配 `references/prompt-existing-project.md`。

## 完成标准

只有同时满足以下条件才报告完成：

- `docs/README.md` 能导航到所有一级分类。
- 治理原则、工作流、分类、命名、状态和模板齐全。
- 功能工作项目录含 `00` 至 `05` 六类文档。
- `docs/04-project-tracking/board.md` 包含 Backlog、Ready、In Progress、Review、Blocked、Done。
- 质量、运维、参考资料和归档目录已建立。
- `AGENTS.md` 含唯一且闭合的受管约束块。
- `--check` 返回成功。
- 最终报告区分：已创建、已更新、已保留、仍需人工补充。

## 参数

```text
--project-root PATH     项目根目录，默认当前目录
--mode auto|new|existing
--project-name NAME     覆盖文档中的项目名称
--owner OWNER           初始工作项负责人，默认 unassigned
--dry-run               只显示计划，不写文件
--check                 只读检查完整性
--force                 覆盖脚本管理的同名文档；需用户明确授权
```

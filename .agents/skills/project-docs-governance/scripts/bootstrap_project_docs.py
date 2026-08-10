#!/usr/bin/env python3
"""Create a non-destructive, docs-first governance system in a project."""

from __future__ import annotations

import argparse
import re
import sys
from datetime import date
from pathlib import Path
from typing import Iterable

MANAGED_START = "<!-- DOCS_GOVERNANCE:START -->"
MANAGED_END = "<!-- DOCS_GOVERNANCE:END -->"
WORK_ITEM = "FEAT-0001-documentation-governance"

REQUIRED_DIRS = (
    "docs/00-governance/templates",
    "docs/01-product/requirements",
    "docs/01-product/roadmap",
    "docs/02-architecture/decisions",
    "docs/02-architecture/data-model",
    "docs/02-architecture/integrations",
    "docs/02-architecture/security",
    f"docs/03-development/features/{WORK_ITEM}",
    "docs/04-project-tracking",
    "docs/05-quality/test-reports",
    "docs/06-operations/deployment",
    "docs/06-operations/runbooks",
    "docs/06-operations/observability",
    "docs/06-operations/incidents",
    "docs/07-reference",
    "docs/08-archive",
)

REQUIRED_FILES = (
    "docs/README.md",
    "docs/00-governance/documentation-principles.md",
    "docs/00-governance/documentation-workflow.md",
    "docs/00-governance/classification-and-hierarchy.md",
    "docs/00-governance/naming-and-versioning.md",
    "docs/00-governance/status-and-lifecycle.md",
    "docs/00-governance/templates/feature-spec.md",
    "docs/00-governance/templates/technical-design.md",
    "docs/00-governance/templates/implementation-plan.md",
    "docs/00-governance/templates/implementation-log.md",
    "docs/00-governance/templates/test-report.md",
    "docs/00-governance/templates/release-and-handoff.md",
    "docs/00-governance/templates/adr.md",
    "docs/01-product/requirements/README.md",
    "docs/01-product/roadmap/README.md",
    "docs/02-architecture/README.md",
    "docs/02-architecture/decisions/README.md",
    "docs/02-architecture/data-model/README.md",
    "docs/02-architecture/integrations/README.md",
    "docs/02-architecture/security/README.md",
    "docs/03-development/README.md",
    f"docs/03-development/features/{WORK_ITEM}/00-feature-spec.md",
    f"docs/03-development/features/{WORK_ITEM}/01-technical-design.md",
    f"docs/03-development/features/{WORK_ITEM}/02-implementation-plan.md",
    f"docs/03-development/features/{WORK_ITEM}/03-implementation-log.md",
    f"docs/03-development/features/{WORK_ITEM}/04-test-report.md",
    f"docs/03-development/features/{WORK_ITEM}/05-release-and-handoff.md",
    "docs/04-project-tracking/board.md",
    "docs/04-project-tracking/milestones.md",
    "docs/04-project-tracking/risks-and-blockers.md",
    "docs/04-project-tracking/project-changelog.md",
    "docs/05-quality/test-strategy.md",
    "docs/05-quality/quality-gates.md",
    "docs/05-quality/test-reports/README.md",
    "docs/06-operations/deployment/README.md",
    "docs/06-operations/runbooks/README.md",
    "docs/06-operations/observability/README.md",
    "docs/06-operations/incidents/README.md",
    "docs/07-reference/glossary.md",
    "docs/07-reference/external-references.md",
    "docs/08-archive/README.md",
)

SOURCE_DIRS = {"src", "app", "apps", "packages", "lib", "server", "client", "backend", "frontend", "cmd", "internal"}
BUILD_FILES = {
    "package.json", "pnpm-lock.yaml", "yarn.lock", "package-lock.json", "pyproject.toml",
    "requirements.txt", "go.mod", "Cargo.toml", "pom.xml", "build.gradle", "build.gradle.kts",
    "composer.json", "Gemfile", "mix.exs", "CMakeLists.txt", "Makefile",
}


def metadata(title: str, status: str, owner: str, item_id: str = "FEAT-0001") -> str:
    today = date.today().isoformat()
    return f"""# {title}

| 字段 | 值 |
| --- | --- |
| status | {status} |
| owner | {owner} |
| last_updated | {today} |
| related_item | {item_id} |
| version | v1.0.0 |
"""


def render_agents_block() -> str:
    return f"""{MANAGED_START}
## 编码智能体文档治理（强制）

### 适用范围与优先级

- 本规则适用于任何代码、配置、依赖、数据结构、接口、部署或运行状态变更。
- 开始工作前先读取 `docs/README.md`、`docs/04-project-tracking/board.md`、相关架构文档和工作项文档。
- 本规则与更具体目录中的 `AGENTS.md` 冲突时，以范围更具体且不降低质量/安全门槛的规则为准。
- 纯问答和只读分析无需创建工作项；一旦产生项目变更，必须进入文档闭环。

### 开发前：无文档不得编码

1. 为工作分配 ID：`FEAT-xxxx`、`BUG-xxxx`、`CHORE-xxxx` 或 `SEC-xxxx`。
2. 创建 `docs/03-development/features/<ID>-<kebab-case-slug>/`。
3. 至少完成 `00-feature-spec.md`、`01-technical-design.md`、`02-implementation-plan.md`。
4. 规格必须写清范围、非范围、可验证验收标准、依赖、风险与回滚思路；状态达到 `ready` 后才可编码。
5. 将工作项加入 `docs/04-project-tracking/board.md` 并同步为 Ready 或 In Progress。

### 开发中：持续留痕

- 每个有意义的实现阶段更新 `03-implementation-log.md`，记录日期、真实改动、关键命令/结果、决策、偏差、阻塞和下一步。
- 实现偏离规格或设计时，先更新相应文档并说明原因，再继续编码。
- 接口、数据模型、架构、安全、部署方式变化时，同步更新 `docs/02-architecture/` 或 `docs/06-operations/` 对应文档。
- 看板状态变化必须在同一工作周期内更新，Blocked 必须写明原因和解除条件。

### 开发后：完成闭环

1. 更新 `04-test-report.md`：测试环境、命令、用例、结果、失败和未覆盖边界。
2. 更新 `05-release-and-handoff.md`：发布范围、配置/迁移、回滚、已知问题、监控和交接。
3. 同步项目看板、里程碑、风险阻塞和项目变更记录。
4. 只有代码、验证、文档、看板、发布/回滚信息一致，工作项才可进入 Done。

### 文档新增、更新与归档触发条件

- 新功能、缺陷修复、重大重构、安全修复、接口/协议/数据结构变化必须新增或关联工作项文档。
- 需求、范围、实现、测试结果、依赖、风险、发布或回滚变化必须更新文档。
- 文档失效时不得直接删除；移动到 `docs/08-archive/` 或标记 `archived`，写明原因、日期和替代文档。
- 已有文档默认原位保留；未经明确授权不得批量移动、重命名或覆盖。

### 分类、命名与状态

- 分类与层级遵循 `docs/00-governance/classification-and-hierarchy.md`。
- 功能目录使用 `<TYPE>-xxxx-<kebab-case-slug>`；目录内固定使用 `00` 至 `05` 文档序列。
- 状态使用 `draft -> ready -> in_progress -> review -> done -> archived`；临时阻塞使用 `blocked`。
- 看板是状态主事实源；工作项目录是范围、设计、实施与验证细节主事实源。

### 真实性与验证边界

- 只记录实际执行的命令、环境和结果，禁止虚构需求、负责人、日期、测试、线上状态或设备验收。
- 明确区分静态检查、编译/构建、模拟环境、开发运行时、生产环境和真实设备验证。
- 未执行的检查写为“未验证”，并说明原因与风险，不得写成通过。

### PR/提交前检查

- [ ] 工作项规格、设计、计划与实现一致。
- [ ] 实施日志记录了关键改动与偏差。
- [ ] 测试报告含真实证据和未验证边界。
- [ ] 发布/回滚和交接信息已更新。
- [ ] 看板、里程碑、风险和变更记录已同步（如适用）。
- [ ] 新增/更新文档链接有效，无孤立文档。
{MANAGED_END}
"""


def render_common(project: str, owner: str) -> dict[str, str]:
    today = date.today().isoformat()
    item = f"docs/03-development/features/{WORK_ITEM}/"
    return {
        "docs/README.md": f"""# {project} 文档中心

本目录是项目事实、决策、开发过程、质量与运维记录的统一入口。开发遵循“先文档、后开发；过程中留痕；完成后回写”。

## 导航

| 目录 | 内容 | 主事实源 |
| --- | --- | --- |
| `00-governance/` | 文档原则、流程、分类、命名、状态、模板 | 文档治理规则 |
| `01-product/` | 项目目标、路线图、需求 | 产品范围与价值 |
| `02-architecture/` | 系统、数据、接口、安全、ADR | 技术现状与决策 |
| `03-development/` | 开发约定和工作项闭环 | 实现细节与证据 |
| `04-project-tracking/` | 看板、里程碑、风险、变更 | 项目状态 |
| `05-quality/` | 测试策略、报告、质量门禁 | 验证标准与结果 |
| `06-operations/` | 部署、运维、监控、事故 | 运行与恢复 |
| `07-reference/` | 术语和外部资料 | 参考信息 |
| `08-archive/` | 失效和历史文档 | 历史追踪 |

## 开始一个工作项

1. 从 `00-governance/templates/` 复制六类模板到 `03-development/features/<ID>-<slug>/`。
2. 先完成规格、技术设计和实施计划，将状态推进到 `ready`。
3. 在 `04-project-tracking/board.md` 登记并开始实现。
4. 开发中维护实施日志；完成时更新测试报告、发布/交接和看板。

## 真实性边界

模板存在不代表内容完成；构建通过不代表运行时、生产或真实设备已验收。每份报告必须写明实际环境与未验证项。
""",
        "docs/00-governance/documentation-principles.md": """# 文档管理原则

1. **文档优先**：规格和验收标准是编码入口。
2. **事实可追溯**：需求、决策、实现、验证、发布都有路径和日期。
3. **同步更新**：文档不是事后总结，必须与代码和项目状态同周期更新。
4. **单一事实源**：看板管理状态，工作项文档管理细节，ADR 管理长期技术决策。
5. **非破坏维护**：更新优于复制，归档优于删除，保留替代关系。
6. **证据诚实**：只记录实际验证，明确环境和边界。
7. **最小充分**：文档要足以让另一位智能体或开发者继续工作，不堆砌无关内容。

## 完成定义（DoD）

工作项进入 Done 前，代码/配置、验收标准、测试证据、相关架构与运维文档、发布/回滚信息、看板状态必须一致，且无未记录阻塞。
""",
        "docs/00-governance/documentation-workflow.md": """# 文档工作流

## 1. 发现与立项

读取项目索引、看板和相关文档，分配工作项 ID，建立目录，编写规格。

## 2. 设计与计划

补全技术设计和实施计划；跨模块或长期决策创建 ADR。验收标准明确后状态进入 Ready。

## 3. 实施

看板进入 In Progress。按计划编码，持续写实施日志；偏差先回写规格/设计。

## 4. 验证与评审

测试报告记录真实环境、命令、结果和未覆盖边界。看板进入 Review。

## 5. 发布与交接

记录部署、迁移、回滚、监控、已知问题和接手方式；同步变更日志、风险与看板。

## 6. 完成与归档

DoD 全部满足后进入 Done。文档失效时标记 archived 并指向替代文档。
""",
        "docs/00-governance/classification-and-hierarchy.md": """# 文档分类与层级

- `00-governance`：跨项目生命周期的强制规则与模板。
- `01-product`：为什么做、为谁做、何时做。
- `02-architecture`：系统如何工作以及为何这样设计。
- `03-development`：单个工作项从规格到交付的完整过程。
- `04-project-tracking`：跨工作项的进度、风险和变更。
- `05-quality`：验证策略、门禁与证据。
- `06-operations`：如何部署、观察、恢复和复盘。
- `07-reference`：非主事实源的术语和外部资料。
- `08-archive`：已失效但需保留追踪的内容。

文档应放在能回答其主要问题的最低合理层级。需要跨层引用时使用相对链接，不复制完整内容制造多个事实源。
""",
        "docs/00-governance/naming-and-versioning.md": """# 命名与版本规范

## 工作项

- 类型：`FEAT` 功能、`BUG` 缺陷、`CHORE` 工程任务、`SEC` 安全任务。
- 编号：四位数字，在项目内唯一且不复用。
- 目录：`<TYPE>-xxxx-<kebab-case-slug>`。
- 固定文件：`00-feature-spec.md`、`01-technical-design.md`、`02-implementation-plan.md`、`03-implementation-log.md`、`04-test-report.md`、`05-release-and-handoff.md`。

## 元数据

每份工作项文档至少记录 `status`、`owner`、`last_updated`、`related_item`、`version`。日期使用 `YYYY-MM-DD`；未知负责人写 `unassigned`，禁止虚构。

## 版本

重大范围或结构变化提升主版本；新增兼容内容提升次版本；澄清和纠错提升修订版本。Git 历史仍是逐行变更事实源。
""",
        "docs/00-governance/status-and-lifecycle.md": """# 状态与生命周期

`draft -> ready -> in_progress -> review -> done -> archived`

- `draft`：信息不完整，不可编码。
- `ready`：范围、验收、方案和计划足以开工。
- `in_progress`：正在实现并维护日志。
- `review`：实现完成，正在验证或评审。
- `blocked`：任一活动状态暂时受阻，必须记录原因、影响、解除条件。
- `done`：DoD 全部满足。
- `archived`：内容失效，保留原因和替代链接。

状态变化必须同步到看板。不得仅因代码存在就把历史功能标记为 Done。
""",
        "docs/00-governance/templates/feature-spec.md": metadata("<ID> <标题> - 功能规格", "draft", "unassigned", "<ID>") + """
## 1. 背景与问题
## 2. 目标与可衡量结果
## 3. 范围（In Scope）
## 4. 非范围（Out of Scope）
## 5. 用户故事/使用场景
## 6. 验收标准
## 7. 依赖与约束
## 8. 风险与回滚思路
## 9. 待确认事项
""",
        "docs/00-governance/templates/technical-design.md": metadata("<ID> <标题> - 技术设计", "draft", "unassigned", "<ID>") + """
## 1. 当前状态与证据
## 2. 方案概览
## 3. 模块与数据流
## 4. 接口/协议/数据模型
## 5. 异常、安全与权限
## 6. 兼容、迁移与回滚
## 7. 备选方案与取舍
## 8. 测试策略
## 9. 未决问题
""",
        "docs/00-governance/templates/implementation-plan.md": metadata("<ID> <标题> - 实施计划", "draft", "unassigned", "<ID>") + """
## 1. 实施阶段
## 2. 任务清单与顺序
## 3. 影响文件/模块
## 4. 每步验证方式
## 5. 依赖、负责人和阻塞
## 6. 完成与停止条件
""",
        "docs/00-governance/templates/implementation-log.md": metadata("<ID> <标题> - 实施日志", "in_progress", "unassigned", "<ID>") + """
## 日志格式

### YYYY-MM-DD - <阶段>
- 实际改动：
- 命令与结果：
- 设计偏差/决策：
- 阻塞与风险：
- 下一步：
""",
        "docs/00-governance/templates/test-report.md": metadata("<ID> <标题> - 测试报告", "draft", "unassigned", "<ID>") + """
## 1. 验收标准映射
## 2. 测试环境与数据
## 3. 命令/用例/结果/证据
## 4. 缺陷与处理
## 5. 未验证边界
## 6. 结论

> 必须区分静态检查、构建、模拟环境、运行时、生产和真实设备验证。
""",
        "docs/00-governance/templates/release-and-handoff.md": metadata("<ID> <标题> - 发布与交接", "draft", "unassigned", "<ID>") + """
## 1. 交付内容与影响范围
## 2. 配置、数据迁移和兼容性
## 3. 发布步骤与验证
## 4. 回滚触发条件与步骤
## 5. 监控、告警和已知问题
## 6. 未完成项与接手入口
""",
        "docs/00-governance/templates/adr.md": """# ADR-xxxx <决策标题>

| 字段 | 值 |
| --- | --- |
| status | proposed / accepted / superseded / deprecated |
| date | YYYY-MM-DD |
| owners | unassigned |
| related_items | <ID> |

## 背景
## 决策
## 备选方案
## 理由
## 正面与负面影响
## 后续动作
## 替代/被替代关系
""",
        "docs/01-product/requirements/README.md": "# 需求与用户故事\n\n按产品或业务域记录需求，链接到对应工作项；禁止用实现细节替代用户价值和验收标准。\n",
        "docs/01-product/roadmap/README.md": "# 产品路线图\n\n记录阶段目标和优先级。未经确认不得虚构发布日期或承诺。\n",
        "docs/02-architecture/README.md": "# 架构文档\n\n记录系统边界、组件关系、关键数据流，以及 data-model、integrations、security 和 decisions 子目录的入口。\n",
        "docs/02-architecture/decisions/README.md": "# 架构决策记录（ADR）\n\n使用 `ADR-xxxx-<slug>.md`，记录影响多个工作项或长期维护的技术决策。\n",
        "docs/02-architecture/data-model/README.md": "# 数据模型\n\n记录实体、约束、迁移、兼容、索引和数据生命周期。未知内容标记待确认。\n",
        "docs/02-architecture/integrations/README.md": "# 接口与集成\n\n记录内部/外部 API、协议、认证、错误语义、限流、重试和版本兼容。\n",
        "docs/02-architecture/security/README.md": "# 安全架构\n\n记录信任边界、认证授权、敏感数据、威胁、审计及安全验证边界。\n",
        "docs/03-development/README.md": """# 开发工作项

所有产生项目变更的工作都必须建立可追踪工作项。目录位于 `features/<TYPE>-xxxx-<slug>/`，使用治理模板中的 `00` 至 `05` 六文档闭环。历史代码不自动视为已补档。
""",
        f"{item}00-feature-spec.md": metadata("FEAT-0001 文档治理初始化 - 功能规格", "done", owner) + """
## 背景与目标

建立适用于编码智能体协作的文档优先治理框架，使后续开发过程可追踪、可验证、可交接。

## 范围

- 建立 `docs/` 分类与层级。
- 建立工作项模板、项目看板、质量和运维入口。
- 将新增、更新、使用和归档规则写入 `AGENTS.md` 受管块。

## 非范围

- 不开发业务功能。
- 不替项目虚构需求、架构、历史状态或验证结果。
- 不迁移、覆盖或删除既有文档。

## 验收标准

1. 文档目录与模板齐全。
2. 看板存在并包含完整状态列。
3. AGENTS.md 受管块唯一且闭合。
4. 只读完整性检查通过。
""",
        f"{item}01-technical-design.md": metadata("FEAT-0001 文档治理初始化 - 技术设计", "done", owner) + """
## 方案

采用分层目录、固定工作项六文档、项目看板和 AGENTS.md 受管规则。初始化脚本默认只创建缺失文件；重复运行只刷新自身受管规则。

## 关键边界

- 看板是状态主事实源。
- 工作项目录是实现细节主事实源。
- ADR 是长期技术决策主事实源。
- 自动生成的现状基线只声明目录级发现，不冒充架构审计。
""",
        f"{item}02-implementation-plan.md": metadata("FEAT-0001 文档治理初始化 - 实施计划", "done", owner) + """
1. 判断新项目或已有项目模式。
2. 创建缺失目录和治理文档。
3. 创建模板和治理初始化工作项。
4. 增量合并 AGENTS.md 受管规则。
5. 执行只读完整性检查并报告保留边界。
""",
        f"{item}03-implementation-log.md": metadata("FEAT-0001 文档治理初始化 - 实施日志", "done", owner) + f"""
## {today} - 初始化

- 创建文档分类、模板、看板、质量和运维入口。
- 增量写入 AGENTS.md 文档治理受管块。
- 未修改业务代码；既有同名文档默认保留。
""",
        f"{item}04-test-report.md": metadata("FEAT-0001 文档治理初始化 - 测试报告", "done", owner) + """
## 已验证

- 必需目录与文件存在性检查。
- AGENTS.md 受管标记唯一性和闭合性检查。
- 看板状态列存在性检查。

## 验证边界

本报告只证明文档框架完整，不证明业务文档内容、业务功能、运行时、生产环境或真实设备已验收。
""",
        f"{item}05-release-and-handoff.md": metadata("FEAT-0001 文档治理初始化 - 发布与交接", "done", owner) + """
## 交付

- 文档治理目录、模板、项目跟踪入口和 AGENTS.md 约束。

## 使用

下一个真实工作项从 `00-feature-spec.md` 开始，先写清验收标准，再进入设计、计划和编码。

## 回滚

移除本次新增的文档前必须确认其中没有后续人工内容；AGENTS.md 仅移除受管标记块。默认不建议删除，应归档并保留历史。
""",
        "docs/04-project-tracking/board.md": f"""# 项目进度看板

> 状态主事实源。状态变化时即时更新。最近初始化：{today}。

## Backlog

- [ ] **FEAT-0002** <首个真实业务功能/历史文档盘点>
  - owner: unassigned
  - priority: 待确认
  - dependencies: 待确认
  - target_date: 待确认
  - blocker: 无
  - docs: `docs/03-development/features/FEAT-0002-<slug>/`
  - last_updated: {today}

## Ready

- 当前无。

## In Progress

- 当前无。

## Review

- 当前无。

## Blocked

- 当前无。

## Done

- [x] **FEAT-0001** 文档治理初始化
  - owner: {owner}
  - priority: P0
  - dependencies: 无
  - target_date: {today}
  - blocker: 无
  - docs: `docs/03-development/features/{WORK_ITEM}/`
  - last_updated: {today}
""",
        "docs/04-project-tracking/milestones.md": """# 项目里程碑

| 里程碑 | 目标 | 关联工作项 | 状态 | 目标日期 |
| --- | --- | --- | --- | --- |
| M0 | 建立文档治理 | FEAT-0001 | done | 已完成 |
| M1 | <待确认> | <待确认> | draft | <待确认> |
""",
        "docs/04-project-tracking/risks-and-blockers.md": """# 风险与阻塞

| ID | 类型 | 描述 | 影响 | 负责人 | 缓解/解除条件 | 状态 | 更新时间 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| RISK-0001 | 流程 | 文档与真实实现不同步 | 交付不可追踪 | unassigned | 每个工作周期执行 DoD 检查 | open | <YYYY-MM-DD> |
""",
        "docs/04-project-tracking/project-changelog.md": f"""# 项目变更记录

## {today}

- 建立文档优先治理框架（FEAT-0001）。
""",
        "docs/05-quality/test-strategy.md": """# 测试策略

每个工作项将验收标准映射到测试层级。按项目需要覆盖静态检查、单元、集成、端到端、性能、安全、兼容、生产冒烟和真实设备；不适用或未执行时写明原因。

测试报告必须记录环境、版本、命令/步骤、输入、预期、实际结果、证据、失败和未验证边界。
""",
        "docs/05-quality/quality-gates.md": """# 质量门禁

- 规格、设计和计划达到 ready。
- 实现与文档一致，偏差有记录。
- 验收标准有真实验证证据。
- 风险、安全、兼容和回滚已评估。
- 测试报告明确未验证边界。
- 发布/交接、看板和变更记录已同步。

具体 lint、typecheck、build、test、E2E 和生产验收命令应按项目技术栈补充，不得猜测。
""",
        "docs/05-quality/test-reports/README.md": "# 跨工作项测试报告\n\n存放版本、系统级或跨工作项测试报告；单工作项验证仍记录在其 `04-test-report.md`。\n",
        "docs/06-operations/deployment/README.md": "# 部署文档\n\n记录环境、前置条件、配置、密钥边界、部署、迁移、验证和回滚。不得写入真实秘密。\n",
        "docs/06-operations/runbooks/README.md": "# 运维手册\n\n记录可执行的诊断、恢复、升级和降级步骤，以及权限和升级联系人。\n",
        "docs/06-operations/observability/README.md": "# 可观测性\n\n记录日志、指标、链路、告警、SLO 及验证方式。未知阈值标记待确认。\n",
        "docs/06-operations/incidents/README.md": "# 故障复盘\n\n按日期和事件编号记录时间线、影响、根因、恢复、证据和行动项；避免归咎个人。\n",
        "docs/07-reference/glossary.md": "# 术语表\n\n| 术语 | 含义 | 来源/备注 |\n| --- | --- | --- |\n| DoD | Definition of Done，完成定义 | 文档治理 |\n| ADR | Architecture Decision Record，架构决策记录 | `docs/02-architecture/decisions/` |\n",
        "docs/07-reference/external-references.md": "# 外部参考资料\n\n记录标题、链接、版本/访问日期、用途和适用范围。外部资料不是本项目状态主事实源。\n",
        "docs/08-archive/README.md": "# 文档归档\n\n失效文档保留原始内容，并在顶部注明 `archived`、归档日期、原因和替代文档链接。未经明确授权不得删除历史。\n",
    }


def visible_entries(root: Path) -> list[str]:
    ignored = {".git", ".next", "node_modules", "vendor", "dist", "build", ".venv", "venv", "__pycache__"}
    return sorted(p.name for p in root.iterdir() if p.name not in ignored)


def detect_mode(root: Path) -> tuple[str, list[str]]:
    evidence: list[str] = []
    entries = visible_entries(root)
    for name in entries:
        path = root / name
        if name in BUILD_FILES:
            evidence.append(f"发现构建/依赖文件 `{name}`")
        if path.is_dir() and name in SOURCE_DIRS:
            evidence.append(f"发现源码目录 `{name}/`")
    git = root / ".git"
    if git.exists():
        evidence.append("发现版本库 `.git`")
    mode = "existing" if evidence else "new"
    return mode, evidence or ["未发现常见源码目录、构建清单或版本库"]


def render_mode_files(root: Path, project: str, mode: str) -> dict[str, str]:
    if mode == "new":
        return {
            "docs/01-product/project-charter.md": f"""# {project} 项目章程

## 项目愿景
<待确认>

## 目标用户与核心问题
<待确认>

## 可衡量目标
<待确认>

## 初始范围与非范围
<待确认>

## 约束、干系人与决策权限
<待确认>
""",
            "docs/02-architecture/system-overview.md": """# 系统架构总览

当前为新项目占位文档。确定需求与技术选型后，记录系统上下文、模块边界、数据流、运行环境、关键质量属性和对应 ADR；不得提前虚构。
""",
        }

    entries = visible_entries(root)
    build = [name for name in entries if name in BUILD_FILES]
    source = [name + "/" for name in entries if (root / name).is_dir() and name in SOURCE_DIRS]
    docs = []
    docs_root = root / "docs"
    if docs_root.exists():
        for path in sorted(docs_root.rglob("*.md")):
            rel = path.relative_to(root).as_posix()
            if rel not in REQUIRED_FILES and WORK_ITEM not in rel:
                docs.append(rel)
            if len(docs) >= 50:
                break

    def bullets(items: list[str], empty: str) -> str:
        return "\n".join(f"- `{item}`" for item in items) if items else f"- {empty}"

    return {
        "docs/02-architecture/current-state-baseline.md": f"""# {project} 当前状态基线

> 本文由目录级只读盘点生成，只代表自动发现，不等同于完整架构审计。内容需在真实开发中持续校正。

## 自动发现：构建与依赖清单

{bullets(build, '未发现常见构建清单，待人工确认')}

## 自动发现：主要源码目录

{bullets(source, '未发现常见源码目录，待人工确认')}

## 初始化前已有 Markdown 文档（最多列 50 项）

{bullets(docs, '未发现治理框架以外的 Markdown 文档，或需人工登记')}

## 待人工确认

- 项目目标、用户、业务域和当前优先级。
- 系统入口、模块边界、数据流和外部集成。
- 测试、部署、生产环境、监控和真实验收状态。
- 历史功能与现有文档的对应关系；不得根据代码存在推断为 Done。
""",
    }


def normalize(content: str) -> str:
    return content.rstrip() + "\n"


def validate_agents_markers(root: Path) -> None:
    path = root / "AGENTS.md"
    if not path.exists():
        return
    text = path.read_text(encoding="utf-8")
    start_count = text.count(MANAGED_START)
    end_count = text.count(MANAGED_END)
    if start_count != end_count or start_count > 1:
        raise RuntimeError("AGENTS.md 中的文档治理受管标记不唯一或未闭合，请人工处理后重试")


def upsert_agents(root: Path, dry_run: bool) -> str:
    path = root / "AGENTS.md"
    block = normalize(render_agents_block())
    if not path.exists():
        if not dry_run:
            path.write_text("# AGENTS.md\n\n" + block, encoding="utf-8")
        return "created"

    text = path.read_text(encoding="utf-8")
    start_count = text.count(MANAGED_START)

    if start_count == 1:
        pattern = re.compile(re.escape(MANAGED_START) + r".*?" + re.escape(MANAGED_END), re.DOTALL)
        updated = pattern.sub(block.rstrip(), text)
        status = "kept" if normalize(updated) == normalize(text) else "updated-managed-block"
    else:
        updated = normalize(text) + "\n" + block
        status = "appended-managed-block"
    if not dry_run and status != "kept":
        path.write_text(normalize(updated), encoding="utf-8")
    return status


def write_files(root: Path, files: dict[str, str], force: bool, dry_run: bool) -> dict[str, list[str]]:
    report = {"created": [], "updated": [], "kept": []}
    for rel, content in sorted(files.items()):
        path = root / rel
        if path.exists() and not force:
            report["kept"].append(rel)
            continue
        status = "updated" if path.exists() else "created"
        report[status].append(rel)
        if not dry_run:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_text(normalize(content), encoding="utf-8")
    return report


def check_project(root: Path) -> list[str]:
    errors: list[str] = []
    for rel in REQUIRED_DIRS:
        if not (root / rel).is_dir():
            errors.append(f"缺少目录: {rel}")
    for rel in REQUIRED_FILES:
        if not (root / rel).is_file():
            errors.append(f"缺少文件: {rel}")

    mode_files = (
        root / "docs/01-product/project-charter.md",
        root / "docs/02-architecture/current-state-baseline.md",
    )
    if not any(path.is_file() for path in mode_files):
        errors.append("缺少模式专属基线: project-charter.md 或 current-state-baseline.md")

    agents = root / "AGENTS.md"
    if not agents.is_file():
        errors.append("缺少文件: AGENTS.md")
    else:
        text = agents.read_text(encoding="utf-8")
        if text.count(MANAGED_START) != 1 or text.count(MANAGED_END) != 1:
            errors.append("AGENTS.md 文档治理受管标记必须唯一且闭合")

    board = root / "docs/04-project-tracking/board.md"
    if board.is_file():
        text = board.read_text(encoding="utf-8")
        for heading in ("Backlog", "Ready", "In Progress", "Review", "Blocked", "Done"):
            if f"## {heading}" not in text:
                errors.append(f"看板缺少状态列: {heading}")
    return errors


def print_list(title: str, items: Iterable[str]) -> None:
    values = list(items)
    print(f"{title} ({len(values)})")
    for value in values:
        print(f"  - {value}")


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="初始化或检查项目文档治理框架")
    parser.add_argument("--project-root", default=".", help="项目根目录，默认当前目录")
    parser.add_argument("--mode", choices=("auto", "new", "existing"), default="auto")
    parser.add_argument("--project-name", help="项目展示名称")
    parser.add_argument("--owner", default="unassigned", help="初始工作项负责人")
    parser.add_argument("--dry-run", action="store_true", help="只显示写入计划")
    parser.add_argument("--check", action="store_true", help="只读检查完整性")
    parser.add_argument("--force", action="store_true", help="覆盖脚本管理的同名文档")
    return parser.parse_args()


def main() -> int:
    args = parse_args()
    root = Path(args.project_root).expanduser().resolve()
    if not root.is_dir():
        print(f"错误：项目根目录不存在或不是目录: {root}", file=sys.stderr)
        return 2
    if args.check and (args.dry_run or args.force):
        print("错误：--check 不能与 --dry-run 或 --force 同时使用", file=sys.stderr)
        return 2

    try:
        validate_agents_markers(root)
    except (OSError, UnicodeError, RuntimeError) as exc:
        print(f"错误：{exc}", file=sys.stderr)
        return 2

    if args.check:
        errors = check_project(root)
        if errors:
            print_list("检查失败", errors)
            return 1
        print(f"检查通过: {root}")
        return 0

    detected, evidence = detect_mode(root)
    mode = detected if args.mode == "auto" else args.mode
    project = args.project_name or root.name
    print(f"project_root={root}")
    print(f"mode={mode} (detected={detected})")
    print_list("模式证据", evidence)
    if args.dry_run:
        print("dry_run=true")

    if not args.dry_run:
        for rel in REQUIRED_DIRS:
            (root / rel).mkdir(parents=True, exist_ok=True)

    files = render_common(project, args.owner)
    files.update(render_mode_files(root, project, mode))
    report = write_files(root, files, force=args.force, dry_run=args.dry_run)
    agents_status = upsert_agents(root, dry_run=args.dry_run)
    print_list("已创建/计划创建", report["created"])
    print_list("已更新/计划更新", report["updated"])
    print_list("原样保留", report["kept"])
    print(f"AGENTS.md={agents_status}")

    if args.dry_run:
        print("预览完成：未写入任何文件。")
        return 0

    errors = check_project(root)
    if errors:
        print_list("完整性检查失败", errors)
        return 1
    print("完整性检查通过。待人工补充：项目目标、真实架构、历史状态、项目专属质量门禁。")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())

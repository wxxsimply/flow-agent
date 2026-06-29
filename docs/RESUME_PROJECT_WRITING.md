# 项目经历编写指南（林炜豪简历模板）

本文档基于 [林炜豪简历-模板.docx](林炜豪简历-模板.docx) 归纳「项目经历」写法，供 [resume_content.py](../scripts/resume_content.py) 与 Word 简历生成使用。FlowAgent 长文素材见 [RESUME_PROJECT.md](RESUME_PROJECT.md)。

---

## 1. 区块结构

整份简历中与项目相关的顺序（模板）：

1. 个人信息 + 页眉横线
2. 竞赛获奖（无「获奖经历」标题，直接列表）
3. **实习经历**（可选）：公司介绍 + `主要职责：` + 编号细节
4. **项目经历**（区块标题含横线装饰）→ 一个或多个项目
5. 专业技能
6. 教育经历

单个项目的段落顺序（推荐**分段落**，与空白模板填充逻辑一致）：

```
项目经历                    ← 区块标题（文字 + 横线）
项目名    角色    时间       ← 三 Tab 项目头
项目简介段落                 ← 场景 + 技术栈 + 可验证结果
项目地址：…                  ← 固定前缀
项目亮点：                   ← 或 主要工作： / 主要职责：
1.主题：…                    ← 编号要点（多条）
2.主题：…
…
[可选] 空行横线              ← 多项目之间的分隔
```

---

## 2. 两种项目头

| 样式 | 格式 | 适用 |
|------|------|------|
| **段落三 Tab** | `项目名\t角色\t时间` | 单项目、课程项目、当前 FlowAgent |
| **三列表格** | 列：项目名 \| 角色 \| 时间 | 模板中短链、DeepResearch 等多产品项目 |

伍孝轩简历采用**段落三 Tab**，与 `fill_project_section` 一致。

---

## 3. 简介写作公式

```
[业务场景] + [核心架构/技术栈] + [可验证结果]
```

**模板示例（高并发短链）：**

> 设计面向短信营销与电商场景的高性能短链平台；通过 Gin+Redis+MySQL+GRPC+ETCD 支撑短链生成/重定向，2 核 4G 压测性能达 1000+ QPS 写入、6000+ QPS 读取。

**要点：**

- 第一句交代**为谁、解决什么**
- 第二句点**关键技术栈**
- 尽量给出**可核对的结果**（QPS、延迟、存储规模、阶段数、覆盖率等），勿编造无依据的数字

---

## 4. 标签选择

| 标签 | 使用场景 |
|------|----------|
| **项目亮点：** | 产品型 / 自研 Agent / 有业务或工程亮点的项目（短链、DeepResearch、**FlowAgent**） |
| **主要工作：** | 课程/学习型项目（如 MIT 6.5840 分布式 KV） |
| **主要职责：** | **实习经历**中的职责摘要行（非项目经历） |

FlowAgent 为自研 AIGC Agent 产品，应使用 **`项目亮点：`**。

---

## 5. 分点规范

- **编号**：`N.主题词：` — **数字与主题之间不加空格**（与专业技能 `1. ` 区分）
- **结构**：`动作 + 技术/模块 + 效果`
- **长度**：约 70–100 字/条；一页简历建议 5–6 条
- **内容**：至少 2 条含可验证信息（阶段数、命令、降级策略、测试范围等）

**模板示例：**

```
1.使用"校验码+分布式布隆过滤器"，实现无效重定向请求过滤，解决缓存穿透
2.水平分表将数据库分为若干分表，…可达568亿存储
```

---

## 6. 项目地址

统一前缀 **`项目地址：`**，后接：

- 公开仓库：`https://github.com/<user>/<repo>`
- 暂未开源：`flow-agent（本地 monorepo 项目）` 等说明

---

## 7. FlowAgent 应用示例

### 优化前（过短、标签不符）

- 标签：`主要工作：`
- 简介：有场景与流水线，缺结果收束
- 分点：~50 字，效果句不足

### 优化后（对齐模板，事实不变）

**项目头：**

```
FlowAgent（AI微电影/短视频全自动生产Agent）	独立开发	2025年05月-至今
```

**简介：**

> 该项目是基于 Go 自研的全链路 AIGC Agent，面向 网文连载 → 分镜 → AI 音画合成 → 合规发布 场景，将扩写、分镜、制片、合规、发布拆为可恢复、可验收的阶段流水线；支持 CLI 与 Desktop 双入口，产物落盘 runs/<run_id>/artifacts/，可通过 flowagent resume --from-stage 断点续跑。基于 YAML 阶段机编排七段流水线，产物可验收、可断点续跑。

**项目地址：**

```
项目地址：flow-agent（本地 monorepo 项目）
```

**标签：** `项目亮点：`

**分点（示例）：**

```
1.工作流引擎：基于 YAML 配置编排 Plan→Publish 七阶段流水线；设计 duration_ok 等阶段门禁与 --auto-gate，支持万相/Seedance 等多 Provider 按成本切换。
2.分镜 Agent：实现首镜扩写为结构化分镜 JSON；按阶段注入 Director Skills，驱动 LLM 与 i2v prompt，并自动审查修复 physics_cues 与 held_props。
3.多模态制片：编排 TTS+文生图+i2v+FFmpeg 合成流水线；Provider 欠费/403 时自动降级 DashScope+Ken Burns，避免单点 API 失败中断整片。
4.视频质量优化：集成 physics_cues 与 forbidden_physics 约束；PropLock 按镜注入与跨镜 held_props 继承，修复道具穿模与左右手瞬移。
5.连载记忆与成本：SQLite FTS 构建 SeriesVault 知识库供 Continuity 检索；cost-ledger 真实用量统计与 flowagent cost 命令归档。
6.工程化交付：提供 flowagent CLI 与 Vue3 Desktop Web UI；核心模块 go test 覆盖 prop_lock、skills registry、shot continuity 等。
```

---

## 8. 与生成脚本

| 文件 | 作用 |
|------|------|
| [scripts/resume_content.py](../scripts/resume_content.py) | 唯一内容源（`ResumeData` / `default_resume()`） |
| [scripts/fill_resume_from_template.py](../scripts/fill_resume_from_template.py) | 写入空白 Word 模板（`fill_project_section`） |
| [scripts/build_resume_docx.py](../scripts/build_resume_docx.py) | 一键生成 `docs/伍孝轩-Go后端简历-new.docx` |

修改项目文案后执行：

```bash
cd scripts && python build_resume_docx.py
```

---

## 9. 与 KV 课程项目的差异

模板中 MIT 6.5840 KV 项目将标题、简介、地址、标签合并在**同一段落**（Word 软换行）。伍孝轩采用**分段落**排版，可读性更好，生成脚本保持分段，不强制合并为 mega-paragraph。

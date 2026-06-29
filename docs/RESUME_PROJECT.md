# 伍孝轩

手机号：13727480563 | 邮箱：3110337564@qq.com  
28届本科 | Golang | AI 应用 / AIGC 工作流  
求职意向：互联网 | 期望城市：广州 / 深圳 / 东莞

---

## 获奖情况

2025年 ACM-CCPC 福建省大学生程序设计竞赛全国邀请赛 金奖  
2025年 ACM-ICPC 国际大学生程序设计竞赛武汉站全国邀请赛 银奖  
2026年 ACM-CCPC 中国大学生程序设计竞赛广西站全国邀请赛 银奖  
2025年 ACM-CCPC 中国大学生程序设计竞赛东北站全国邀请赛 铜奖  
2025年 ACM-CCPC 广东省大学生程序设计竞赛邀请赛 铜奖  
2026年 ACM-CCPC 广东省大学生程序设计竞赛邀请赛 铜奖  
2025年 蓝桥杯 省级一等奖  
2025年 蓝桥杯 全国三等奖  
2026年 团体程序设计竞赛 团队全国三等奖  
2026年 团体程序设计竞赛 个人全国三等奖  
2025年 团体程序设计竞赛 团队全国三等奖  

---

## 项目经历

### Flow Agent（AI 微电影 / 短视频全自动生产 Agent） 独立开发 2025年05月 - 至今

该项目是基于 Go 自研的全链路 AIGC Agent，面向 **网文连载 → 分镜 → AI 音画合成 → 合规发布** 场景，将扩写、分镜、制片、合规、发布拆为可恢复、可验收的阶段流水线；支持 CLI 与 Desktop 双入口，产物落盘 `runs/<run_id>/artifacts/`，可通过 `flowagent resume --from-stage` 断点续跑。

**主要职责：**

1. **工作流引擎**：基于 YAML 配置实现 Plan / Write / Continuity / Storyboard / Produce / Compliance / Publish 全阶段编排（`internal/workflow`、`internal/runner`），设计阶段门禁（duration_ok、no_block_issues、cost_budget）与 `--auto-gate` 开发模式；支持多种 stack（万相 flash、Seedance、economy 等）按成本与质量切换 Provider。
2. **分镜 Agent**：实现 Shot Language Expander，将用户第一镜扩写为结构化分镜 JSON；设计 Director Skills 体系（`.cursor/skills/` + `internal/agent/skills/registry.go`），按阶段注入镜头语言、物理 realism、道具锁定等 reference，驱动 LLM 与 i2v prompt；实现分镜审查与自动修复（physics_cues、held_props、action_beats）。
3. **多模态制片**：编排 TTS + 文生图 + 图生视频（Seedance / Wan i2v）+ FFmpeg 合成流水线，支持软/硬镜间衔接与并行制片；实现 Provider 欠费 / 403 时自动降级为 DashScope 图/TTS + Ken Burns 动效（`produce_state.go`、`media_fallback.go`），避免整片因单点 API 失败而中断。
4. **视频质量优化**：集成 PhyT2V / PhysVid / VideoPhy 思路的正向 physics_cues 与 forbidden_physics 约束，可选 WMReward BoN 多候选选优；定位并修复 i2v 道具左右手瞬移、消失、变形问题，实现 PropLockBlock 按镜注入、跨镜 held_props 继承及关键帧 beat 与 motion prompt 握姿对齐。
5. **连载记忆与成本**：基于 SQLite FTS 实现 SeriesVault 连载知识库，供 Continuity / Planner 检索设定与角色状态；实现真实用量 cost-ledger 与 `flowagent cost` 命令，支持 metrics 回流与 Learn 归档。
6. **工程化交付**：提供 `cmd/flowagent` CLI 与 `cmd/flowagent-desktop`（Vue 3 + Vite）本地 Web UI；核心模块单元测试覆盖 prop_lock、skills registry、shot continuity 等（`go test ./pkg/artifacts/... ./internal/agent/...`）。

---

## 专业技能

熟练使用 Go 进行后端与 Agent 系统开发，掌握 goroutine 并发、errgroup 并行任务、接口抽象与模块化设计，具备独立排查与调试能力；熟悉数据结构与算法，有多项 ACM/ICPC 竞赛获奖经历。

熟悉 LLM 应用工程化：Prompt 分层与调优、结构化 JSON 输出、按阶段注入 Skill Reference；了解 RAG 思路（SeriesVault FTS 检索）、多 Agent 协作与产物驱动工作流设计。

有大模型与 AIGC 多模态实践经验：DeepSeek、阿里云百炼 DashScope（通义 TTS / 万相 t2i·i2v）、火山方舟 Seedream/Seedance；熟悉 TTS、文生图、图生视频调用与 FFmpeg 音视频合成。

熟悉 YAML 工作流、阶段状态机、Provider 插件化与降级容错设计；了解 SQLite 存储与全文检索；能使用 Vue 3 搭建简单 Desktop Web UI。

掌握 Linux 与 Git 基本操作，能独立完成项目本地运行、配置与协作开发；注重可运行、可验收、可降级，而非 demo 级一次性脚本。

---

## 教育经历

**东莞理工学院** 2024年09月 - 2028年06月  
大数据技术与应用 本科  

主修课程：算法与数据结构、计算机网络、操作系统、数据库原理等。

"""Canonical resume text (from 伍孝轩-Go后端简历.docx)."""

from __future__ import annotations

from dataclasses import dataclass, field


@dataclass
class ResumeData:
    name: str
    phone: str
    email: str
    wechat: str
    tagline: str
    awards: list[str] = field(default_factory=list)
    project_title: str = ""
    project_role: str = ""
    project_period: str = ""
    project_intro: str = ""
    project_url: str = ""
    project_label: str = "项目亮点："
    project_duties: list[str] = field(default_factory=list)
    skills: list[str] = field(default_factory=list)
    edu_school: str = ""
    edu_major: str = ""


def default_resume() -> ResumeData:
    return ResumeData(
        name="伍孝轩",
        phone="13727480563",
        email="3110337564@qq.com",
        wechat="wxx13798608302",
        tagline="28届本科|Golang|AI应用/AIGC工作流",
        awards=[
            "2025年ACM/CCPC福建省大学生程序设计竞赛全国邀请赛金奖",
            "2025年ACM/CCPC国际大学生程序设计竞赛武汉站全国邀请赛银奖",
            "2026年ACM/CCPC中国大学生程序设计竞赛广西站全国邀请赛银奖",
            "2025年ACM/CCPC中国大学生程序设计竞赛东北站全国邀请赛铜奖",
            "2025年ACM/CCPC广东省大学生程序设计竞赛邀请赛铜奖",
            "2026年ACM/CCPC广东省大学生程序设计竞赛邀请赛铜奖",
            "2025年蓝桥杯全国三等奖",
            "2026年团体程序设计竞赛团队全国三等奖",
            "2026年团体程序设计竞赛个人全国三等奖",
            "2025年团体程序设计竞赛团队全国三等奖",
        ],
        project_title="FlowAgent（AI微电影/短视频全自动生产Agent）",
        project_role="独立开发",
        project_period="2025年05月-至今",
        project_intro=(
            "该项目是基于 Go 自研的全链路 AIGC Agent，面向 网文连载 → 分镜 → AI 音画合成 → 合规发布 场景，"
            "将扩写、分镜、制片、合规、发布拆为可恢复、可验收的阶段流水线；支持 CLI 与 Desktop 双入口，"
            "产物落盘 runs/<run_id>/artifacts/，可通过 flowagent resume --from-stage 断点续跑。"
            "基于 YAML 阶段机编排七段流水线，产物可验收、可断点续跑。"
        ),
        project_url="项目地址：flow-agent（本地 monorepo 项目）",
        project_label="项目亮点：",
        project_duties=[
            "1.工作流引擎：YAML 七阶段编排，阶段门禁与多 Provider 切换。",
            "2.分镜 Agent：首镜 JSON 扩写，Skills 注入与分镜自动审查修复。",
            "3.多模态制片：TTS+文生图+i2v+FFmpeg 合成，Provider 故障自动降级。",
            "4.视频质量优化：physics 约束与 PropLock，修复道具穿模与跨镜一致性。",
            "5.连载记忆与成本：SQLite FTS 知识库与 cost-ledger 用量统计归档。",
            "6.工程化交付：CLI/Desktop 双入口，核心模块单元测试覆盖。",
        ],
        skills=[
            "1. 熟练 Go 后端与 Agent 开发，掌握 goroutine、errgroup 模块化设计。",
            "2. 熟悉 LLM 工程化：Prompt 调优、结构化 JSON 与 Skill 注入。",
            "3. 熟悉 AIGC 多模态：DashScope/Seedance，TTS、文生图与图生视频。",
            "4. 熟悉 YAML 工作流、状态机、Provider 插件化与 SQLite 全文检索。",
            "5. 熟悉并发编程：channel、条件变量与协程。",
            "6. 熟悉 Linux/Git，可独立部署服务器与协同开发。",
            "7. 熟悉数据结构算法：树、图、搜索、DP 等，有竞赛基础。",
            "8. 熟悉计算机网络与操作系统。",
            "9. 热爱新技术，抗压强，擅长 AI Agent/MCP/Skill 辅助开发。",
        ],
        edu_school="东莞理工学院（一本）\t2024年9月-2028年6月",
        edu_major="大数据技术与应用",
    )

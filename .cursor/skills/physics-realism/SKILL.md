---

name: physics-realism

description: AI 视频物理可信度：PhyT2V 正向规则、PhysVid/VideoPhy 反事实 negative、WMReward BoN。分镜写 physics_cues/forbidden，produce 注入 motion。

---



# Physics Realism（索引）



与 [micro-movie-director](../micro-movie-director/SKILL.md) 配合：分镜写字段，本 skill 管**成片物理**与**选优**。



## References



| 文件 | 用途 |

|------|------|

| [phyt2v-positive-rules.md](references/phyt2v-positive-rules.md) | **正向** physics_cues：七类规则 + VideoPhy 材质交互 |

| [physvid-negative.md](references/physvid-negative.md) | **负向** forbidden：反事实 negative 句式库 |

| [videophy-material-cues.md](references/videophy-material-cues.md) | 固-固/固-液/液-液 成对表 + 审查触发 |

| [physics-logic.md](../micro-movie-director/references/physics-logic.md) | 分镜主文档（director） |

| [video-generation-forbidden.md](../micro-movie-director/references/video-generation-forbidden.md) | 绝对禁止清单（produce negative） |

| [wmreward-bon.md](references/wmreward-bon.md) | 多候选 i2v + V-JEPA surprise |

| [physics-iq-checklist.md](references/physics-iq-checklist.md) | Physics-IQ 回归 |

| [produce-motion-checklist.md](../micro-movie-director/references/produce-motion-checklist.md) | i2v 短句（MotionPromptBlock） |



## 论文 / GitHub 来源



| 项目 | 贡献 |

|------|------|

| [PhyT2V](https://github.com/pittisl/PhyT2V) | 正向：提取物体+物理规则，CoT 细化 prompt |

| [PhysVid](https://github.com/5aurabhpathak/PhysVid) | 负向：counterfactual negative physics prompts |

| [VideoPhy](https://github.com/Hritikbansal/videophy) | 材质交互 SA/PC 评测维度 |

| [WMReward](https://github.com/facebookresearch/WMReward) | 推理时 surprise 选优 BoN |

| [physics-IQ-benchmark](https://github.com/google-deepmind/physics-IQ-benchmark) | 重力/碰撞/液体/固体/时序回归 |

| [PAR+SDG](https://arxiv.org/abs/2509.24702) | 反事实推理 + 去 implausible 轨迹 |

| [Awesome-Physics-Cognition](https://github.com/minnie-lin/Awesome-Physics-Cognition-based-Video-Generation) | 综述与论文索引 |



## 配置



`config/stacks/micro-movie-wan-flash.yaml` → `video.wmreward_bon`



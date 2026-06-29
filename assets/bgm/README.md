# 微电影情绪 BGM 曲库

将 **免版权** 的 MP3 放入本目录，文件名与情绪标签对应。  
扩写阶段会根据剧情输出 `mood`，Produce 时自动混音（音量见 stack `bgm_volume`）。

## 文件名（必选其一作为回退）

| 文件 | 适用情绪 |
|------|----------|
| `neutral.mp3` | 默认 / 未识别 |
| `tense.mp3` | 紧张 |
| `suspense.mp3` | 悬疑 |
| `sad.mp3` | 悲伤 |
| `warm.mp3` | 温暖治愈 |
| `hopeful.mp3` | 希望励志 |
| `epic.mp3` | 史诗燃 |
| `romantic.mp3` | 浪漫 |
| `horror.mp3` | 恐怖 |
| `comedy.mp3` | 轻松搞笑 |

至少放置 **`neutral.mp3`**，建议按常用情绪多备几首。

## 从网易云 `.ncm` 转换（已购曲目）

项目内已带 `scripts/ncmdump/ncmdump.exe`。在仓库根目录执行：

```powershell
.\scripts\convert-bgm.ps1
```

会把 `assets/bgm/*.ncm` 解密为 FLAC，再转成 `sad.mp3` / `neutral.mp3`（可按 `convert-bgm.ps1` 里的映射表改）。需要本机已安装 **ffmpeg**。

当前曲库映射（可自行调整脚本）：

| 源文件 | 曲库名 |
|--------|--------|
| Tony Greywolf - Feeling Lonely | `sad.mp3` |
| Papa Khan - Diary of a poor kid | `neutral.mp3` |

## 自定义 BGM

```powershell
flowagent run micro-movie --plot "..." --bgm-file D:\music\my-track.mp3
```

## 关闭 BGM

```powershell
flowagent run micro-movie --plot "..." --bgm off
```

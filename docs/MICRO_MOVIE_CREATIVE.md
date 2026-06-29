# 微电影创作选项（2D/3D · 剧情 · 情绪 BGM · 画面约束）

## 1. 自行选择 2D / 3D

```powershell
# 2D 插画 / 赛璐璐（默认）
flowagent run micro-movie --plot "你的完整剧情……" --style 2d --auto-gate

# 3D 动画质感
flowagent run micro-movie --plot "你的完整剧情……" --style 3d --auto-gate

# 单镜预览风格
flowagent test-shot --style 3d --image-prompt "竖屏9:16，女孩站在樱花树下"
```

风格会写入 `artifacts/creative-options.json`，并作用于：

- 万相 **文生图** 的 `[STYLE]` / `[NEG]`
- 万相 **图生视频** 的运镜与物理约束 prompt
- 分镜 LLM 的风格说明

## 2. 自行输入剧情

任选其一（可组合）：

```powershell
# 命令行直接写
flowagent run micro-movie --plot "深夜加班的程序员发现显示器里伸出一只手……" --auto-gate

# 长文本用文件
flowagent run micro-movie --plot-file D:\scripts\my-story.md --auto-gate
```

剧情会保存为 `artifacts/plot-input.md`，经 **expand → script → storyboard** 扩写，不会覆盖你的核心设定。

## 3. 情绪 BGM（与剧情匹配）

1. 扩写阶段输出 `story-spine.json` 中的 `mood` / `tone` / `emotion_arc`
2. 生成 `artifacts/bgm-plan.json`
3. Produce 时从 **`assets/bgm/<mood>.mp3`** 混音（需自备免版权音乐）

```powershell
# 自动按剧情情绪选曲（默认）
flowagent run micro-movie --plot "……" --bgm auto --auto-gate

# 使用自己的 BGM 文件
flowagent run micro-movie --plot "……" --bgm-file D:\music\track.mp3 --auto-gate

# 不要 BGM
flowagent run micro-movie --plot "……" --bgm off --auto-gate
```

曲库说明见 [assets/bgm/README.md](../assets/bgm/README.md)。至少放置 `assets/bgm/neutral.mp3`。

## 4. 减少穿模 / 不合理画面

当前为 **提示词 + 分镜约束**（非 3D 引擎物理模拟），已默认追加：

- 禁止穿模、物体穿透、多余肢体
- 动作幅度适中、单镜单焦点
- 空间与重力逻辑自洽

无法 100% 消除 AI 视频随机性。若某镜失败：

- 用 `--style 2d` 通常比 3D 更稳
- `resume --from-stage storyboard` 重跑分镜
- 单镜先用 `test-shot` 调试 prompt

## 5. 推荐完整命令

```powershell
go build -o bin\flowagent.exe .\cmd\flowagent

flowagent test-shot --style 2d --image-prompt "你的关键画面描述"

flowagent run micro-movie `
  --plot-file .\my-plot.md `
  --style 2d `
  --bgm auto `
  --auto-gate -v
```

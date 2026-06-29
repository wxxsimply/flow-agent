# Physics-IQ 回归评测（可选）：对固定第一镜样例跑 director + produce，并记录产物路径。
# 完整打分需克隆 https://github.com/google-deepmind/physics-IQ-benchmark 并按其 README 安装依赖。
param(
    [string]$OpeningShot = "雨夜霓虹巷口，男子驻足望向橱窗倒影。",
    [string]$Stack = "micro-movie-wan-flash",
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

$bin = Join-Path $root "bin\flowagent.exe"
if (-not (Test-Path $bin)) {
    Write-Host "building flowagent..."
    go build -o $bin ./cmd/flowagent
}

$runDir = Join-Path $root "runs\physics-iq-$(Get-Date -Format 'yyyyMMdd-HHmmss')"
New-Item -ItemType Directory -Force -Path $runDir | Out-Null

$args = @(
    "director",
    "--stack", $Stack,
    "--opening-shot", $OpeningShot,
    "--out", $runDir
)
if ($DryRun) { $args += "--dry-run" }

& $bin @args
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Run artifacts: $runDir"
Write-Host "Storyboard review: $(Join-Path $runDir 'artifacts\storyboard-review.json')"
Write-Host ""
Write-Host "For Physics-IQ scores, clone physics-IQ-benchmark and evaluate generated clips:"
Write-Host "  git clone https://github.com/google-deepmind/physics-IQ-benchmark.git"
Write-Host "  # follow benchmark README on clips under $runDir\clips"

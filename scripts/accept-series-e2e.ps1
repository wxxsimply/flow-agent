# 阶段 J1：同一 series 连续跑第 1～3 集（默认 dry-run，可改 -LiveRun）
param(
    [string]$Series = "demo",
    [string]$Stack = "standard-tier",
    [switch]$LiveRun,
    [switch]$AutoGate
)

$root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $root

# 控制台 UTF-8，避免 flowagent cost 中文乱码（Windows PowerShell 5）
try {
    chcp 65001 | Out-Null
    [Console]::OutputEncoding = [System.Text.UTF8Encoding]::new($false)
    $OutputEncoding = [System.Text.UTF8Encoding]::new($false)
} catch { }

$bin = Join-Path $root "bin\flowagent.exe"
if (-not (Test-Path $bin)) {
    go build -o $bin .\cmd\flowagent
}

$common = @("run", "novel-short-douyin", "--series", $Series, "--stack", $Stack)
if (-not $LiveRun) { $common += "--dry-run" }
if ($AutoGate) { $common += "--auto-gate" }

# flowagent 子进程（如 ffmpeg）可能向 stderr 打 banner；勿用 Stop，否则 PowerShell 误报 NativeCommandError
$prevEAP = $ErrorActionPreference
$ErrorActionPreference = "Continue"

$runIds = @()
$failed = $false
foreach ($ep in 1..3) {
    Write-Host "`n=== Episode $ep ===" -ForegroundColor Cyan
    $script:epRunId = $null
    & $bin @common --episode $ep 2>&1 | ForEach-Object {
        $line = "$_"
        Write-Host $line
        if ($line -match 'run_id=([0-9a-f-]{36})') {
            $script:epRunId = $Matches[1]
        }
    }
    if ($script:epRunId) { $runIds += $script:epRunId }
    $code = $LASTEXITCODE
    if ($code -ne 0) {
        Write-Host "episode $ep failed exit=$code" -ForegroundColor Red
        $failed = $true
        break
    }
}

$ErrorActionPreference = $prevEAP
if ($failed) { exit 1 }

Write-Host "`n=== Summary ===" -ForegroundColor Green
if ($runIds.Count -eq 0) {
    Write-Host "No runs found under runs/ for series=$Series"
    exit 1
}
foreach ($id in $runIds) {
    $runDir = Join-Path $root "runs\$id"
    $m = Get-Content (Join-Path $runDir "manifest.json") -Encoding UTF8 | ConvertFrom-Json
    $ep = $m.episode_no
    $checks = @(
        "artifacts\episode-brief.md",
        "artifacts\chapter.md",
        "artifacts\storyboard.json",
        "artifacts\master.mp4",
        "artifacts\publish-pack.json",
        "artifacts\metrics-snapshot.json"
    )
    Write-Host "ep$ep run_id=$id stage=$($m.stage)"
    foreach ($f in $checks) {
        $ok = Test-Path (Join-Path $runDir $f)
        Write-Host ("  {0} {1}" -f $(if ($ok) { "OK" } else { "MISSING" }), $f)
    }
    if ($LiveRun) {
        & $bin cost --run-id $id 2>&1 | ForEach-Object { "$_" }
    }
}

Write-Host "`nJ1 acceptance done." -ForegroundColor Green
if (-not $LiveRun) {
    Write-Host "For live API run: .\scripts\accept-series-e2e.ps1 -LiveRun -AutoGate"
}

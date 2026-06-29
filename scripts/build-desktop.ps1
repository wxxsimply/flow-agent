# 打包 FlowAgent Studio 桌面 exe（双击弹出 Vue 前端窗口）
param(
    [string]$OutDir = "dist"
)

$ErrorActionPreference = "Stop"
Set-Location (Split-Path $PSScriptRoot -Parent)

Write-Host "Building Vue 3 frontend..."
Push-Location web/ui
npm install
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
npm run build
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Pop-Location

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

Write-Host "Building FlowAgent.exe (WebView2 native window)..."
$env:CGO_ENABLED = "0"
go build -ldflags "-H windowsgui" -o "$OutDir\FlowAgent.exe" ./cmd/flowagent-desktop
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Building flowagent CLI..."
go build -o "$OutDir\flowagent.exe" ./cmd/flowagent
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

$bundle = Join-Path $OutDir "FlowAgent"
New-Item -ItemType Directory -Force -Path $bundle | Out-Null
Copy-Item "$OutDir\FlowAgent.exe" $bundle
foreach ($dir in @("config", "docs", "assets", "ffmpeg")) {
    if (Test-Path $dir) {
        Copy-Item -Recurse -Force $dir (Join-Path $bundle $dir)
    }
}
if (Test-Path "config\providers.local.yaml.example") {
    Copy-Item "config\providers.local.yaml.example" (Join-Path $bundle "config\providers.local.yaml.example")
}

Write-Host ""
Write-Host "Done. Double-click to launch:"
Write-Host "  $bundle\FlowAgent.exe"
Write-Host ""
Write-Host "Dev (from repo root): go run ./cmd/flowagent-desktop"

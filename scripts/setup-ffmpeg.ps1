# 将 FFmpeg 安装到项目根目录 <repo>/ffmpeg/bin/
# 用法: .\scripts\setup-ffmpeg.ps1
# 优先从本机 WinGet 已安装的 Gyan.FFmpeg 复制；否则从 GitHub 下载。

$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root

$dest = Join-Path $Root "ffmpeg"
$bin = Join-Path $dest "bin"
if (Test-Path (Join-Path $bin "ffmpeg.exe")) {
    Write-Host "OK: ffmpeg already at $bin"
    & (Join-Path $bin "ffmpeg.exe") -version | Select-Object -First 1
    exit 0
}

# 1) 从 WinGet 复制（最快）
$packages = Join-Path $env:LOCALAPPDATA "Microsoft\WinGet\Packages"
if (Test-Path $packages) {
    $src = Get-ChildItem $packages -Directory -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -like "*FFmpeg*" } |
        ForEach-Object { Get-ChildItem $_.FullName -Directory -Filter "ffmpeg-*-full_build" -ErrorAction SilentlyContinue } |
        Select-Object -First 1
    if ($src -and (Test-Path (Join-Path $src.FullName "bin\ffmpeg.exe"))) {
        if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
        Copy-Item -Recurse $src.FullName $dest
        Write-Host "Copied from WinGet: $($src.FullName) -> $dest"
        & (Join-Path $bin "ffmpeg.exe") -version | Select-Object -First 1
        exit 0
    }
}

# 2) 下载 essentials（体积较小，约 90MB）
$zip = Join-Path $Root "ffmpeg-download.zip"
$url = "https://github.com/GyanD/codexffmpeg/releases/download/8.1.1/ffmpeg-8.1.1-essentials_build.zip"
Write-Host "Downloading $url ..."
Invoke-WebRequest -Uri $url -OutFile $zip -UseBasicParsing
Expand-Archive -Path $zip -DestinationPath $Root -Force
Remove-Item $zip -Force
$extracted = Get-ChildItem $Root -Directory | Where-Object { $_.Name -like "ffmpeg-*" } | Select-Object -First 1
if ($extracted) {
    if (Test-Path $dest) { Remove-Item -Recurse -Force $dest }
    Rename-Item $extracted.FullName "ffmpeg"
}
if (-not (Test-Path (Join-Path $bin "ffmpeg.exe"))) {
    Write-Error "ffmpeg install failed"
}
Write-Host "OK: $bin"
& (Join-Path $bin "ffmpeg.exe") -version | Select-Object -First 1

# Pack flow-agent for Linux deploy (LF line endings + slim tarball for 2G servers).
param(
    [string]$OutFile = "flow-agent.tar.gz"
)

$Root = Split-Path $PSScriptRoot -Parent
Set-Location $Root

Get-ChildItem -Path (Join-Path $Root "scripts") -Filter "*.sh" | ForEach-Object {
    $text = [IO.File]::ReadAllText($_.FullName)
    $text = $text -replace "`r`n", "`n" -replace "`r", "`n"
    $utf8NoBom = New-Object System.Text.UTF8Encoding $false
    [IO.File]::WriteAllText($_.FullName, $text, $utf8NoBom)
    Write-Host "LF: $($_.Name)"
}

if (Test-Path $OutFile) { Remove-Item $OutFile -Force }

# 与 .gitignore 对齐：排除大目录，适合 2G/2核服务器 scp 上传
tar -czf $OutFile `
    --exclude=runs `
    --exclude=series `
    --exclude=data `
    --exclude=dist `
    --exclude=ffmpeg `
    --exclude=bin `
    --exclude=node_modules `
    --exclude=web/ui/node_modules `
    --exclude=web/dist `
    --exclude=.git `
    --exclude=.docker `
    --exclude=.idea `
    --exclude=.vscode `
    --exclude=.cursor/plans `
    --exclude=*.tar.gz `
    --exclude=*.log `
    --exclude=*.mp4 `
    --exclude=*.exe `
    --exclude=$OutFile `
    .

Write-Host ""
Write-Host "Created: $Root\$OutFile"
Write-Host "Upload:  scp $OutFile root@<服务器IP>:/opt/"

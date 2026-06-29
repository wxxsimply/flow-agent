# Convert NetEase .ncm in assets/bgm to mood-named MP3 for flow-agent.
# Requires: scripts/ncmdump/ncmdump.exe, ffmpeg on PATH.
param(
    [string]$BgmDir = (Join-Path $PSScriptRoot "..\assets\bgm"),
    [switch]$KeepFlac
)

$ErrorActionPreference = "Stop"
$root = Resolve-Path (Join-Path $PSScriptRoot "..")
$bgm = Resolve-Path $BgmDir
$ncmdump = Join-Path $PSScriptRoot "ncmdump\ncmdump.exe"

if (-not (Test-Path $ncmdump)) {
    Write-Error "ncmdump not found at $ncmdump — download ncmdump 1.5+ into scripts/ncmdump/"
}
$ffmpeg = (Get-Command ffmpeg -ErrorAction SilentlyContinue)?.Source
if (-not $ffmpeg) { Write-Error "ffmpeg not found on PATH" }

& $ncmdump -d $bgm -o $bgm

$map = @{
    "*Feeling Lonely*" = "sad.mp3"
    "*Diary of a poor kid*" = "neutral.mp3"
}

Get-ChildItem $bgm -Filter *.flac | ForEach-Object {
    $out = $null
    foreach ($pat in $map.Keys) {
        if ($_.Name -like $pat) { $out = Join-Path $bgm $map[$pat]; break }
    }
    if (-not $out) {
        Write-Warning "No mood mapping for $($_.Name), skipping"
        return
    }
    & $ffmpeg -y -i $_.FullName -map 0:a -codec:a libmp3lame -qscale:a 2 $out
    Write-Host "-> $out"
    if (-not $KeepFlac) { Remove-Item $_.FullName -Force }
}

Write-Host "Done. MP3 files:"
Get-ChildItem $bgm -Filter *.mp3 | Format-Table Name, @{N='MB';E={[math]::Round($_.Length/1MB,2)}}

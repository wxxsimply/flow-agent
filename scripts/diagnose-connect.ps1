# Connectivity diagnostic for deployment server -> debug-ef1997.log (NDJSON)
param(
    [string]$TargetHost = "39.102.136.31",
    [int[]]$Ports = @(22, 8080),
    [string]$LogFile = "debug-ef1997.log"
)

$sessionId = "ef1997"
$logPath = Join-Path (Split-Path $PSScriptRoot -Parent) $LogFile

function Write-DebugLog {
    param([string]$HypothesisId, [string]$Location, [string]$Message, [hashtable]$Data)
    $entry = @{
        sessionId    = $sessionId
        hypothesisId = $HypothesisId
        location     = $Location
        message      = $Message
        data         = $Data
        timestamp    = [DateTimeOffset]::UtcNow.ToUnixTimeMilliseconds()
    } | ConvertTo-Json -Compress
    Add-Content -Path $logPath -Value $entry -Encoding UTF8
}

Write-Host "==> FlowAgent connectivity diagnostic: $TargetHost"
Write-Host "    log: $logPath"

$routes = Get-NetRoute -DestinationPrefix "172.19.0.0/16" -ErrorAction SilentlyContinue
Write-DebugLog "H1" "diagnose-connect.ps1:routes" "route_to_172_19" @{
    hasRoute = ($null -ne $routes -and $routes.Count -gt 0)
    count    = @($routes).Count
}

$ping = Test-Connection -ComputerName $TargetHost -Count 2 -Quiet -ErrorAction SilentlyContinue
Write-DebugLog "H2" "diagnose-connect.ps1:ping" "ping_result" @{ pingOk = [bool]$ping }

foreach ($port in $Ports) {
    $tcp = Test-NetConnection -ComputerName $TargetHost -Port $port -WarningAction SilentlyContinue
    $hid = if ($port -eq 22) { "H3" } else { "H4" }
    Write-DebugLog $hid "diagnose-connect.ps1:tcp" "tcp_port_test" @{
        port      = $port
        tcpOk     = [bool]$tcp.TcpTestSucceeded
        pingOk    = [bool]$tcp.PingSucceeded
        interface = $tcp.InterfaceAlias
    }
}

$tracert = tracert -d -h 10 -w 800 $TargetHost 2>&1 | Out-String
$reached = $tracert -match [regex]::Escape($TargetHost)
Write-DebugLog "H5" "diagnose-connect.ps1:tracert" "tracert_summary" @{
    reachedTarget = [bool]$reached
    preview       = ($tracert -split "`n" | Select-Object -First 12) -join " | "
}

Write-Host ""
Write-Host "Summary:"
Write-Host "  route 172.19.0.0/16: $(if ($routes) { 'YES' } else { 'NO' })"
Write-Host "  ping: $(if ($ping) { 'OK' } else { 'FAIL' })"
foreach ($port in $Ports) {
    $t = Test-NetConnection -ComputerName $TargetHost -Port $port -WarningAction SilentlyContinue
    Write-Host "  TCP ${port}: $(if ($t.TcpTestSucceeded) { 'OK' } else { 'FAIL' })"
}

if (-not $ping) {
    Write-Host ""
    Write-Host "FIX: Your PC cannot reach $TargetHost (routing/VPN/firewall)."
    Write-Host "  1) Check cloud security group allows inbound TCP 22 and 8080"
    Write-Host "  2) Use cloud console Web Terminal on the server"
    Write-Host "  3) Confirm public IP (e.g. 39.102.136.31) not private/VPN-only address"
    Write-Host "  4) Try non-default SSH port if applicable"
}

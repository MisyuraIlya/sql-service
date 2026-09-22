<#
.SYNOPSIS
    Build sql-service and deploy it to the DigiTradeService Windows service (NSSM).

.DESCRIPTION
    Order of operations - nothing on the production box is touched until the
    build succeeded AND the service is confirmed stopped:

      1. build  service.exe into a staging folder (skipped with -SkipBuild/-ExePath)
      2. stop   the NSSM service
      3. back up the live service.exe into .\backups\
      4. copy   the new exe into place
      5. start  the service, wait for RUNNING and for the port to listen
      6. on any failure after step 3: restore the backup and start it again

.EXAMPLE
    powershell -ExecutionPolicy Bypass -File .claude\skills\deploy\deploy.ps1

.EXAMPLE
    # deploy a binary built elsewhere (e.g. the GitHub Actions artifact)
    powershell -ExecutionPolicy Bypass -File .claude\skills\deploy\deploy.ps1 -ExePath C:\Temp\service.exe
#>
[CmdletBinding()]
param(
    [string] $RepoPath    = 'C:\Services\sql-service',
    [string] $ServiceRoot = 'C:\Services\DigiTradeService',
    [string] $ServiceName = 'DigiTradeService',
    [int]    $Port        = 9952,
    [string] $ExePath     = '',
    [switch] $SkipBuild,
    [int]    $KeepBackups = 10,
    [int]    $TimeoutSec  = 60
)

$ErrorActionPreference = 'Stop'

# Fatal before anything in production has been touched: print the message plainly
# (no PowerShell stack trace) and leave the running service alone.
function Die($msg) {
    if ($staging -and (Test-Path $staging)) { Remove-Item $staging -Recurse -Force -ErrorAction SilentlyContinue }
    Write-Host "ERROR: $msg" -ForegroundColor Red
    exit 1
}

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Warn($msg) { Write-Host "WARN: $msg"  -ForegroundColor Yellow }
function Write-Ok  ($msg) { Write-Host "OK:   $msg"  -ForegroundColor Green }

function Get-GoExe {
    $cmd = Get-Command go -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }
    $candidates = @(
        (Join-Path $env:ProgramFiles 'Go\bin\go.exe'),
        (Join-Path ${env:ProgramFiles(x86)} 'Go\bin\go.exe'),
        (Join-Path $env:LOCALAPPDATA 'Programs\Go\bin\go.exe'),
        'C:\Go\bin\go.exe'
    )
    foreach ($c in $candidates) {
        if ($c -and (Test-Path $c)) { return $c }
    }
    return $null
}

# Stop/start the service via the SCM rather than `nssm stop|start`. NSSM writes
# routine progress ("Unexpected status SERVICE_START_PENDING in response to
# START control") to stderr, and PowerShell 5.1 under $ErrorActionPreference =
# 'Stop' promotes native-command stderr to a TERMINATING error - so a perfectly
# healthy start looked like a failure and triggered a rollback. NSSM still
# supervises the process and applies its configured stop methods either way;
# only the control channel differs.
function Stop-TargetService([string]$Name, [int]$Seconds) {
    Stop-Service -Name $Name -Force -ErrorAction Stop
    return (Wait-ServiceStatus $Name 'Stopped' $Seconds)
}

function Start-TargetService([string]$Name, [int]$Seconds) {
    # Start-Service returns as soon as the SCM accepts the request; START_PENDING
    # is expected, so the poll below - not this call - decides success.
    Start-Service -Name $Name -ErrorAction Stop
    return (Wait-ServiceStatus $Name 'Running' $Seconds)
}

function Wait-ServiceStatus([string]$Name, [string]$Status, [int]$Seconds) {
    $deadline = (Get-Date).AddSeconds($Seconds)
    while ((Get-Date) -lt $deadline) {
        $svc = Get-Service -Name $Name -ErrorAction SilentlyContinue
        if ($svc -and $svc.Status -eq $Status) { return $true }
        Start-Sleep -Milliseconds 500
    }
    return $false
}

function Test-PortListening([int]$P) {
    try {
        return [bool](Get-NetTCPConnection -LocalPort $P -State Listen -ErrorAction SilentlyContinue)
    } catch {
        return $false
    }
}

# ---------------------------------------------------------------- preflight --
$nssm       = Join-Path $ServiceRoot 'nssm.exe'
$liveExe    = Join-Path $ServiceRoot 'service.exe'
$backupDir  = Join-Path $ServiceRoot 'backups'
$logDir     = Join-Path $ServiceRoot 'logs'

if (-not (Test-Path $ServiceRoot)) { Die "Service folder not found: $ServiceRoot" }
if (-not (Test-Path $nssm))        { Die "nssm.exe not found: $nssm" }
if (-not (Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
    Die "Service '$ServiceName' is not installed. Run $ServiceRoot\install-service.ps1 from an elevated PowerShell first."
}
if (-not (Test-Path $backupDir)) { New-Item -ItemType Directory -Path $backupDir | Out-Null }

$principal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Warn 'This shell is NOT elevated. Stopping/starting the service will most likely fail with Access Denied.'
    Write-Warn 'If it does, re-run this script from an Administrator PowerShell. Nothing is modified before the stop succeeds.'
}

# -------------------------------------------------------------------- build --
$staging = Join-Path ([IO.Path]::GetTempPath()) ("sql-service-deploy-" + (Get-Date -Format 'yyyyMMdd_HHmmss'))
$newExe  = ''

if ($ExePath) {
    if (-not (Test-Path $ExePath)) { Die "-ExePath not found: $ExePath" }
    $newExe = (Resolve-Path $ExePath).Path
    Write-Step "Using prebuilt binary: $newExe"
} elseif ($SkipBuild) {
    Die '-SkipBuild requires -ExePath <path to an already built service.exe>.'
} else {
    if (-not (Test-Path (Join-Path $RepoPath 'go.mod'))) { Die "No go.mod in $RepoPath - is that the repo?" }

    $go = Get-GoExe
    if (-not $go) {
        Die @'
Go toolchain not found on this machine.
Either install Go (https://go.dev/dl/ - the msi puts go.exe in C:\Program Files\Go\bin),
or build elsewhere and deploy that binary with:
    deploy.ps1 -ExePath <path to service.exe>
The repo's GitHub Actions workflow (build-windows.yml) publishes exactly such a service.exe.
'@
    }

    New-Item -ItemType Directory -Path $staging | Out-Null
    $newExe = Join-Path $staging 'service.exe'

    Write-Step "Building with $go"
    Push-Location $RepoPath
    try {
        $env:GOOS = 'windows'; $env:GOARCH = 'amd64'; $env:CGO_ENABLED = '0'
        & $go build -trimpath -ldflags '-s -w' -o $newExe ./cmd
        $buildExit = $LASTEXITCODE
    } finally {
        Pop-Location
    }
    if ($buildExit -ne 0)          { Die "go build failed (exit $buildExit) - nothing was deployed." }
    if (-not (Test-Path $newExe))  { Die 'go build reported success but produced no binary.' }
    Write-Ok ("Built {0:N0} bytes" -f (Get-Item $newExe).Length)
}

# git provenance, best effort
$rev = ''
try {
    Push-Location $RepoPath
    $rev = (& git rev-parse --short HEAD 2>$null)
    $dirty = (& git status --porcelain 2>$null)
    if ($dirty) { $rev = "$rev (working tree has uncommitted changes)" }
    Pop-Location
} catch { }
if ($rev) { Write-Host "    source: $rev" }

# --------------------------------------------------------------------- stop --
Write-Step "Stopping $ServiceName"
try {
    $stopped = Stop-TargetService $ServiceName $TimeoutSec
} catch {
    Die "Could not stop ${ServiceName}: $($_.Exception.Message) Nothing was changed - the old binary is still in place and running. If this was Access Denied, re-run from an elevated PowerShell."
}
if (-not $stopped) {
    $st = (Get-Service $ServiceName).Status
    Die "Service did not stop within ${TimeoutSec}s (status: $st). Nothing was changed - the old binary is still in place and running. If this was Access Denied, re-run from an elevated PowerShell."
}
Write-Ok 'Service stopped'

# ------------------------------------------------------------ backup + copy --
$backup = ''
if (Test-Path $liveExe) {
    $backup = Join-Path $backupDir ('service_' + (Get-Date -Format 'yyyy-MM-dd_HHmmss') + '.exe')
    Copy-Item $liveExe $backup -Force
    Write-Ok "Backed up current binary -> $backup"
} else {
    Write-Warn "No existing $liveExe to back up (first deploy?)"
}

$deployFailed = $false
try {
    Write-Step "Copying new binary -> $liveExe"
    Copy-Item $newExe $liveExe -Force

    Write-Step "Starting $ServiceName"
    if (-not (Start-TargetService $ServiceName $TimeoutSec)) {
        throw "Service did not reach Running within ${TimeoutSec}s."
    }

    Write-Step "Waiting for port $Port to listen"
    $deadline = (Get-Date).AddSeconds($TimeoutSec)
    $listening = $false
    while ((Get-Date) -lt $deadline) {
        if (Test-PortListening $Port) { $listening = $true; break }
        if ((Get-Service $ServiceName).Status -ne 'Running') { break }
        Start-Sleep -Milliseconds 500
    }
    if (-not $listening) { throw "Service is running but nothing is listening on port $Port." }
    Write-Ok "Listening on port $Port"
}
catch {
    $deployFailed = $true
    Write-Host "DEPLOY FAILED: $($_.Exception.Message)" -ForegroundColor Red

    # show why, before rolling back
    $errLog = Get-ChildItem (Join-Path $logDir 'service.err*.log') -ErrorAction SilentlyContinue |
              Sort-Object LastWriteTime -Descending | Select-Object -First 1
    if ($errLog) {
        Write-Host "--- tail of $($errLog.Name) ---" -ForegroundColor Red
        Get-Content $errLog.FullName -Tail 20
        Write-Host '--- end ---' -ForegroundColor Red
    }

    # Nothing in here may throw: an exception escaping the rollback would leave
    # the service stopped and production down. Every step is guarded, and the
    # last act is always an attempt to get *something* running again.
    if ($backup) {
        Write-Step 'Rolling back to previous binary'
        try {
            Stop-TargetService $ServiceName $TimeoutSec | Out-Null
            Copy-Item $backup $liveExe -Force
            Write-Ok "Restored $backup"
        } catch {
            Write-Host "Rollback copy failed: $($_.Exception.Message)" -ForegroundColor Red
        }
    } else {
        Write-Host 'No backup existed - restarting on the binary currently in place.' -ForegroundColor Red
    }

    try {
        if (Start-TargetService $ServiceName $TimeoutSec) {
            Write-Ok "$ServiceName is running again"
        } else {
            Write-Host "$ServiceName is DOWN and did not restart. Investigate now." -ForegroundColor Red
        }
    } catch {
        Write-Host "$ServiceName is DOWN - restart failed: $($_.Exception.Message)" -ForegroundColor Red
    }
}

# ------------------------------------------------------------------ cleanup --
if (Test-Path $staging) { Remove-Item $staging -Recurse -Force -ErrorAction SilentlyContinue }

if ($KeepBackups -gt 0) {
    Get-ChildItem (Join-Path $backupDir 'service_*.exe') -ErrorAction SilentlyContinue |
        Sort-Object LastWriteTime -Descending |
        Select-Object -Skip $KeepBackups |
        Remove-Item -Force -ErrorAction SilentlyContinue
}

# ------------------------------------------------------------------ summary --
Write-Host ''
Write-Host '================ deploy summary ================'
$svc = Get-Service $ServiceName
Write-Host ("service   : {0} [{1}]" -f $svc.Name, $svc.Status)
Write-Host ("binary    : {0}" -f $liveExe)
if (Test-Path $liveExe) {
    $f = Get-Item $liveExe
    Write-Host ("            {0:N0} bytes, {1}" -f $f.Length, $f.LastWriteTime)
}
if ($rev)    { Write-Host ("source    : {0}" -f $rev) }
if ($backup) { Write-Host ("backup    : {0}" -f $backup) }
Write-Host ("port      : {0} {1}" -f $Port, $(if (Test-PortListening $Port) { 'listening' } else { 'NOT listening' }))
Write-Host ("result    : {0}" -f $(if ($deployFailed) { 'FAILED (rolled back)' } else { 'SUCCESS' }))
Write-Host '==============================================='

if ($deployFailed) { exit 1 }
exit 0

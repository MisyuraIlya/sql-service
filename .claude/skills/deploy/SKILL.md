---
name: deploy
description: Build sql-service and deploy it to production on this server - go build service.exe, back up the live binary, copy it into C:\Services\DigiTradeService, and restart the DigiTradeService NSSM service. Use when the user says deploy, ship, release, publish, "put it on prod", or asks to restart/roll back the DigiTradeService service.
---

# Deploy sql-service to production

This server **is** the production box. The repo at `C:\Services\sql-service` is built into
`service.exe` and dropped into `C:\Services\DigiTradeService`, where NSSM runs it as the
Windows service `DigiTradeService` on port **9952**.

| | |
|---|---|
| repo | `C:\Services\sql-service` |
| entrypoint | `./cmd` (module `sql-service`) |
| deploy folder | `C:\Services\DigiTradeService` |
| live binary | `C:\Services\DigiTradeService\service.exe` |
| service manager | `C:\Services\DigiTradeService\nssm.exe` |
| service name | `DigiTradeService` |
| port | `9952` |
| config | `C:\Services\DigiTradeService\.env` (DB creds - **never** overwrite or print it) |
| logs | `C:\Services\DigiTradeService\logs\service.{out,err}*.log` |
| backups | `C:\Services\DigiTradeService\backups\service_<timestamp>.exe` |

## How to deploy

Everything is in `deploy.ps1` next to this file. Run it from the repo root:

```bash
powershell -NoProfile -ExecutionPolicy Bypass -File .claude/skills/deploy/deploy.ps1
```

It builds, stops, backs up, copies, starts, verifies, and rolls back on failure - in that
order, so **nothing in production is touched until the build succeeds and the service is
confirmed stopped**. A failed build or a failed stop leaves the running service untouched.

Then report the summary block it prints (status, binary size/time, source commit, backup
path, port). Exit code 0 = deployed, 1 = failed and rolled back.

### Before running it

1. **Confirm with the user** before the first deploy of a session. This restarts a live
   service - a few seconds of downtime on port 9952. Don't deploy as a side effect of some
   other task; only when asked.
2. Check the working tree (`git status`, `git log --oneline -3`) and tell the user what is
   about to ship, especially if there are uncommitted changes - the script deploys the
   working tree, not the last commit.
3. If the code changed in this session, make sure it compiles first: `go build ./...`.

### Useful flags

| flag | use |
|---|---|
| `-ExePath <path>` | deploy a binary built elsewhere instead of building here (see below) |
| `-KeepBackups <n>` | how many backups to retain, default 10 |
| `-TimeoutSec <n>` | stop/start/port wait, default 60 |
| `-ServiceName`, `-ServiceRoot`, `-RepoPath`, `-Port` | override the defaults above |

## Two things that will bite you on this machine

**Go is not installed here.** The build step fails with an explicit message. Either install
Go from https://go.dev/dl/ (the msi puts `go.exe` in `C:\Program Files\Go\bin`), or build the
binary elsewhere and deploy that:

```bash
powershell -NoProfile -ExecutionPolicy Bypass -File .claude/skills/deploy/deploy.ps1 -ExePath C:/Temp/service.exe
```

The repo's `.github/workflows/build-windows.yml` already produces exactly this binary as the
`service-windows-amd64` artifact on every push to `main` (and attaches it to `v*` tags), so
downloading that is a legitimate source for `-ExePath`.

**The Claude Code shell is not elevated.** `nssm stop/start` needs Administrator rights;
without them the script aborts at the stop step *before changing anything* and says so. When
that happens, ask the user to run it themselves from an elevated terminal, e.g. by typing:

```
! powershell -NoProfile -ExecutionPolicy Bypass -File C:\Services\sql-service\.claude\skills\deploy\deploy.ps1
```

`nssm status DigiTradeService` and `Get-Service` are read-only and work unelevated - use them
to check state without touching production. (NSSM prints UTF-16, so its output looks
space-separated; prefer `Get-Service DigiTradeService` when you need to parse the status.)

## If it breaks

The script rolls back automatically and prints the tail of the newest `service.err*.log`. To
roll back by hand to any earlier build:

```powershell
$root = 'C:\Services\DigiTradeService'
Stop-Service DigiTradeService -Force
Copy-Item $root\backups\service_<timestamp>.exe $root\service.exe -Force
Start-Service DigiTradeService
```

Use `Stop-Service`/`Start-Service`, **not** `nssm stop`/`nssm start`, for service
control in any script. NSSM reports routine progress such as `Unexpected status
SERVICE_START_PENDING in response to START control` on **stderr**, and PowerShell 5.1 under
`$ErrorActionPreference = 'Stop'` promotes native-command stderr into a terminating error -
so a healthy start reads as a failure. That exact trap took production down once: it faked a
failed start, triggered a rollback, and then threw again inside the rollback handler, which
abandoned the service in the Stopped state. NSSM still supervises the process and applies its
configured stop methods; only the control channel differs.

Common causes of a service that starts and then dies: bad or missing `.env` (the app calls
`log.Fatalf` when the DB connection fails), or port 9952 already held by an orphaned
`service.exe` - check with `Get-NetTCPConnection -LocalPort 9952 -State Listen`.

Never commit `service.exe`, `.env`, or anything from `backups\` / `logs\` to the repo.

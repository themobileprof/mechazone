# Mechazone shop installer for Windows. Run once in PowerShell:
#   Right-click install.ps1 → Run with PowerShell
$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $Root

function Need($cmd, $hint) {
  if (-not (Get-Command $cmd -ErrorAction SilentlyContinue)) {
    throw "Missing $cmd. $hint"
  }
}

Write-Host ""
Write-Host "==> Mechazone installer"
Write-Host "    This sets up the bay app on this computer. Once only."

Need python "Install Python 3 from https://www.python.org/downloads/ (tick Add to PATH)."
Need node "Install Node.js LTS from https://nodejs.org/"
Need go "Install Go 1.24 or newer from https://go.dev/dl/"
Need git "Install Git from https://git-scm.com/"

$goVer = (go version) -replace '.*go([0-9]+)\.([0-9]+).*', '$1.$2'
$goMajor, $goMinor = $goVer.Split('.')
if ([int]$goMajor -lt 1 -or ([int]$goMajor -eq 1 -and [int]$goMinor -lt 24)) {
  throw "Go $goVer is too old. Install Go 1.24+ from https://go.dev/dl/"
}

if (-not (Get-Command psql -ErrorAction SilentlyContinue)) {
  Write-Host "    PostgreSQL (psql) was not found. Install it from https://www.postgresql.org/download/windows/"
  Write-Host "    Then create a database named mechazone in pgAdmin and run this installer again."
  throw "PostgreSQL is required."
}

$exists = & psql -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='mechazone'" 2>$null
if ($exists -notmatch "1") {
  & createdb mechazone
}

if (-not (Test-Path "$Root\.env")) { Copy-Item "$Root\.env.example" "$Root\.env" }
$ui = "$Root\client\dist" -replace '\\', '/'
$envText = Get-Content "$Root\.env" -Raw
if ($envText -match "UI_DIR=") {
  $envText = $envText -replace "UI_DIR=.*", "UI_DIR=$ui"
} else {
  $envText += "`nUI_DIR=$ui`n"
}
Set-Content -Path "$Root\.env" -Value $envText -NoNewline

Write-Host "==> Worker"
python -m venv "$Root\diagnostic-worker\.venv"
& "$Root\diagnostic-worker\.venv\Scripts\python.exe" -m pip install -q --upgrade pip
& "$Root\diagnostic-worker\.venv\Scripts\python.exe" -m pip install -q -r "$Root\diagnostic-worker\requirements.txt"

Write-Host "==> Shop screen"
Set-Location "$Root\client"
if (Test-Path package-lock.json) { npm ci --no-fund --no-audit } else { npm install --no-fund --no-audit }
npm run build
Set-Location $Root

Write-Host "==> Ledger"
New-Item -ItemType Directory -Force -Path "$Root\bin", "$Root\var" | Out-Null
Set-Location "$Root\cloud-backend"
$env:GOTOOLCHAIN = "local"
go build -o "$Root\bin\mechazone-server.exe" .\cmd\server
Set-Location $Root

$Wsh = New-Object -ComObject WScript.Shell
$desk = [Environment]::GetFolderPath("Desktop")
$lnk = $Wsh.CreateShortcut("$desk\Mechazone.lnk")
$lnk.TargetPath = "powershell.exe"
$lnk.Arguments = "-WindowStyle Hidden -ExecutionPolicy Bypass -File `"$Root\scripts\start-mechazone.ps1`""
$lnk.WorkingDirectory = $Root
$lnk.Save()

Write-Host ""
Write-Host "Done. Double-click Mechazone on the Desktop."
Write-Host "First login: admin@mechazone.local"
Write-Host "First password: change-me-now"
Write-Host "Guide: docs\install.md"

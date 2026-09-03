$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Set-Location $Root
New-Item -ItemType Directory -Force -Path "$Root\var" | Out-Null
if (Test-Path "$Root\.env") {
  Get-Content "$Root\.env" | ForEach-Object {
    if ($_ -match '^\s*#' -or $_ -notmatch '=') { return }
    $k, $v = $_.Split('=', 2)
    [Environment]::SetEnvironmentVariable($k.Trim(), $v.Trim(), "Process")
  }
}
$env:UI_DIR = if ($env:UI_DIR) { $env:UI_DIR } else { "$Root\client\dist" }
$server = "$Root\bin\mechazone-server.exe"
if (-not (Test-Path $server)) { throw "Run install.ps1 first. See docs\install.md." }
Start-Process -WindowStyle Hidden -FilePath $server -WorkingDirectory $Root
$py = "$Root\diagnostic-worker\.venv\Scripts\python.exe"
if (Test-Path $py) {
  Start-Process -WindowStyle Hidden -FilePath $py -ArgumentList "-m", "mechazone_worker" -WorkingDirectory "$Root\diagnostic-worker"
}
Start-Sleep -Seconds 2
Start-Process "http://127.0.0.1:8080"

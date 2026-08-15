$Root = Split-Path -Parent (Split-Path -Parent $MyInvocation.MyCommand.Path)
Get-Process mechazone-server -ErrorAction SilentlyContinue | Stop-Process -Force
Get-CimInstance Win32_Process -Filter "Name = 'python.exe'" -ErrorAction SilentlyContinue |
  Where-Object { $_.CommandLine -and $_.CommandLine -like "*mechazone_worker*" } |
  ForEach-Object { Stop-Process -Id $_.ProcessId -Force -ErrorAction SilentlyContinue }
Write-Host "Mechazone stopped."

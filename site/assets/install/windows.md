Open **PowerShell** (search for *PowerShell* in the Start menu), paste this line, and press Enter:

```powershell
$a = if ($env:PROCESSOR_ARCHITECTURE -eq 'ARM64') { 'arm64' } else { 'amd64' }; $d = "$env:LOCALAPPDATA\Programs\wopr"; New-Item -ItemType Directory -Force $d | Out-Null; curl.exe -fLo "$d\wopr.exe" "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_windows_$a.exe"; if ($LASTEXITCODE -eq 0) { $p = [Environment]::GetEnvironmentVariable('Path', 'User'); if (";$p;" -notlike "*;$d;*") { [Environment]::SetEnvironmentVariable('Path', "$p;$d", 'User') }; & "$d\wopr.exe" }
```

It puts `wopr.exe` in a folder of your own, adds that folder to your `PATH`, and starts it. To play again,
open a new PowerShell window and type `wopr`.

Open **PowerShell** (search for *PowerShell* in the Start menu), paste this line, and press Enter:

```powershell
$a = if ("$env:PROCESSOR_ARCHITECTURE $env:PROCESSOR_ARCHITEW6432" -match 'ARM64') { 'arm64' } else { 'amd64' }; $d = "$env:LOCALAPPDATA\Programs\wopr"; $null = New-Item -ItemType Directory -Force $d; curl.exe -fL --ssl-revoke-best-effort -o "$d\wopr.exe" "https://github.com/GhostofGoes/WOPR/releases/download/v@VERSION@/wopr_@VERSION@_windows_$a.exe"; if ($LASTEXITCODE -eq 0) { $k = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment'); $p = $k.GetValue('Path', '', 'DoNotExpandEnvironmentNames'); if (($p -split ';') -notcontains $d) { $k.SetValue('Path', ((@($p -split ';') + $d | Where-Object { $_ }) -join ';'), 'ExpandString'); [Environment]::SetEnvironmentVariable('WOPR_PATH_REFRESH', $null, 'User') }; $k.Close(); $env:Path += ";$d"; & "$d\wopr.exe" }
```

It puts `wopr.exe` in `%LOCALAPPDATA%\Programs\wopr`, adds that folder to your `PATH`, and starts it. To
play again, open PowerShell and type `wopr`.

If **Smart App Control** is turned on, Windows blocks `wopr`: it blocks every program that is not signed,
with no exception for one program, and `wopr` is not signed yet.
[Troubleshooting](/usage/troubleshooting) says more.

# Installs wopr_<version>_windows_setup.exe silently for the current user, checks what it installed,
# uninstalls it and checks that it left nothing behind, not even a debug log. Then it does the same over a copy that the
# PowerShell install line put in %LOCALAPPDATA%\Programs\wopr, the same folder, and installs twice.
# Run it on a machine without WOPR installed (a CI runner): it changes the user's PATH while it
# runs, and puts it back at the end.
#
#   pwsh -File packaging/windows/test-installer.ps1 -Installer build/installer/wopr_1.2.3_windows_setup.exe -Version 1.2.3
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Installer,
    [Parameter(Mandatory)] [string] $Version,
    # Where Setup's and Uninstall's logs go; printed when a check fails.
    [string] $LogDir = (Join-Path ([IO.Path]::GetTempPath()) 'wopr-installer-logs')
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Installer = (Resolve-Path -LiteralPath $Installer).Path
$null = New-Item -ItemType Directory -Force -Path $LogDir
$app = Join-Path $env:LOCALAPPDATA 'Programs\WOPR'
$oneLinerDir = Join-Path $env:LOCALAPPDATA 'Programs\wopr' # the PowerShell install line's folder
$programs = [Environment]::GetFolderPath('Programs') # the user's Start menu, All apps
$shortcut = Join-Path $programs 'WOPR.lnk'
$desktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'WOPR.lnk'
$debugLogDir = Join-Path $env:LOCALAPPDATA 'wopr' # internal/debuglog's folder: os.UserCacheDir()\wopr
# wopr.iss's AppId, plus the _is1 that Inno Setup adds.
$uninstallKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{A7D86110-58E9-404E-9164-FBDDBDC6AC22}_is1'
$files = 'wopr.exe', 'LICENSE.txt', 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md', 'unins000.exe', 'unins000.dat'
$wantMachine = switch ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64' { 0x8664 }
    'Arm64' { 0xAA64 }
    default { throw "no Windows build for $_" }
}

# The user's PATH as stored: unexpanded, or $null when there is none.
function Get-UserPath {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
    try {
        if ($null -eq $key) { return $null }
        return $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    } finally {
        if ($null -ne $key) { $key.Dispose() }
    }
}

# Untyped, so that $null (no Path value at all) stays $null rather than becoming "".
function Set-UserPath($Value) {
    $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
    try {
        if ($null -eq $Value) {
            $key.DeleteValue('Path', $false)
        } else {
            $key.SetValue('Path', $Value, [Microsoft.Win32.RegistryValueKind]::ExpandString)
        }
    } finally {
        $key.Dispose()
    }
}

# The PATH's entries, without empty ones (Setup may drop a trailing ";").
function Get-PathEntries([AllowNull()] [string] $Path) {
    return @("$Path" -split ';' | Where-Object { $_ -ne '' })
}

# How many PATH entries name the WOPR folder, ignoring case and a trailing backslash, as Windows does.
function Get-PathCount {
    return @(Get-PathEntries (Get-UserPath) | Where-Object { $_.Trim().TrimEnd('\') -eq $app }).Count
}

function Assert([bool] $Condition, [string] $Message) {
    if (-not $Condition) { throw "FAILED: $Message" }
    Write-Host "ok: $Message"
}

function Invoke-Setup([string] $Name) {
    $log = Join-Path $LogDir "$Name.log"
    $p = Start-Process -FilePath $Installer -Wait -PassThru -ArgumentList @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', '/CURRENTUSER', "/LOG=`"$log`"")
    Assert ($p.ExitCode -eq 0) "Setup ($Name) exits with 0, not $($p.ExitCode)"
}

function Invoke-Uninstall([string] $Name) {
    $log = Join-Path $LogDir "$Name.log"
    $p = Start-Process -FilePath (Join-Path $app 'unins000.exe') -Wait -PassThru -ArgumentList @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', "/LOG=`"$log`"")
    Assert ($p.ExitCode -eq 0) "Uninstall ($Name) exits with 0, not $($p.ExitCode)"
    # The uninstaller runs a copy of itself from TEMP, which can still be removing the folder, or
    # retrying the Start menu shortcut (wopr.iss, DeleteStartMenuShortcut), when the first one exits.
    for ($i = 0; $i -lt 60 -and ((Test-Path -LiteralPath $app) -or (Test-Path -LiteralPath $uninstallKey) -or (Test-Path -LiteralPath $shortcut)); $i++) {
        Start-Sleep -Seconds 1
    }
}

function Test-Installed {
    foreach ($f in $files) {
        Assert (Test-Path -LiteralPath (Join-Path $app $f) -PathType Leaf) "$f is in $app"
    }
    $exe = Join-Path $app 'wopr.exe'

    $out = & $exe --version
    Assert ($LASTEXITCODE -eq 0 -and "$out" -like "wopr v$Version (*") "wopr.exe --version prints v${Version}: $out"

    # The PE header's machine field: the build for this computer's CPU.
    $stream = [IO.File]::OpenRead($exe)
    try {
        $reader = [IO.BinaryReader]::new($stream)
        $stream.Position = 0x3C
        $stream.Position = $reader.ReadInt32()
        $signature = $reader.ReadUInt32()
        $machine = $reader.ReadUInt16()
    } finally {
        $stream.Dispose()
    }
    Assert ($signature -eq 0x4550 -and $machine -eq $wantMachine) ('wopr.exe is for this CPU: machine 0x{0:X4}, want 0x{1:X4}' -f $machine, $wantMachine)

    Assert (Test-Path -LiteralPath $shortcut -PathType Leaf) "the Start menu shortcut is $shortcut"
    # Read the shortcut, then let go of the COM objects at once rather than at some later garbage
    # collection, so that this test never holds the file Uninstall must delete.
    $wsh = New-Object -ComObject WScript.Shell
    $link = $wsh.CreateShortcut($shortcut)
    try {
        $target, $arguments, $workingDir = $link.TargetPath, $link.Arguments, $link.WorkingDirectory
    } finally {
        [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($link)
        [void][Runtime.InteropServices.Marshal]::FinalReleaseComObject($wsh)
    }
    Assert ($target -eq $exe -and $arguments -eq '') "the shortcut runs $exe with no arguments: $target $arguments"
    Assert ($workingDir -eq $app) "the shortcut starts in ${app}: $workingDir"
    Assert (-not (Test-Path -LiteralPath (Join-Path $programs 'WOPR') -PathType Container)) 'there is no WOPR folder in the Start menu'
    Assert (-not (Test-Path -LiteralPath $desktopShortcut)) 'there is no desktop shortcut (that task is off by default)'

    Assert (Test-Path -LiteralPath $uninstallKey) "Settings > Apps lists WOPR ($uninstallKey)"
    $entry = Get-ItemProperty -LiteralPath $uninstallKey
    Assert ($entry.DisplayName -eq 'WOPR') "its name is WOPR: $($entry.DisplayName)"
    Assert ($entry.DisplayVersion -eq $Version) "its version is ${Version}: $($entry.DisplayVersion)"
    Assert ($entry.Publisher -eq 'Christopher Goes') "its publisher is Christopher Goes: $($entry.Publisher)"
    Assert ($entry.DisplayIcon -eq $exe) "its icon is wopr.exe's: $($entry.DisplayIcon)"

    Assert ((Get-PathCount) -eq 1) "PATH has $app once: $(Get-UserPath)"
    $kind = (Get-Item -LiteralPath 'HKCU:\Environment').GetValueKind('Path')
    Assert ($kind -eq 'ExpandString') "PATH is still REG_EXPAND_SZ: $kind"
}

function Test-Removed {
    Assert (-not (Test-Path -LiteralPath $app)) "$app is gone"
    Assert (-not (Test-Path -LiteralPath $shortcut)) 'the Start menu shortcut is gone'
    Assert (-not (Test-Path -LiteralPath $uninstallKey)) 'Settings > Apps no longer lists WOPR'
    Assert ((Get-PathCount) -eq 0) "PATH no longer has ${app}: $(Get-UserPath)"
    Assert (-not (Test-Path -LiteralPath $debugLogDir)) "the debug log and its folder, $debugLogDir, are gone"
}

# A debug log, as WOPR_DEBUG=1 writes it, for Uninstall to delete.
function New-DebugLog {
    $null = New-Item -ItemType Directory -Force -Path $debugLogDir
    Set-Content -LiteralPath (Join-Path $debugLogDir 'debug.log') -Value 'a debug log'
}

$originalPath = Get-UserPath
Assert (-not (Test-Path -LiteralPath $app) -and -not (Test-Path -LiteralPath $uninstallKey) -and (Get-PathCount) -eq 0) 'WOPR is not installed yet'
try {
    Write-Host "== A first install, with the default tasks"
    Invoke-Setup 'install'
    Test-Installed
    New-DebugLog
    Invoke-Uninstall 'uninstall'
    Test-Removed
    Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'PATH is back to what it was, every other entry untouched'

    Write-Host "== Over the PowerShell install line's copy, then again over itself"
    $null = New-Item -ItemType Directory -Force -Path $oneLinerDir
    Set-Content -LiteralPath (Join-Path $oneLinerDir 'wopr.exe') -Value 'an older wopr.exe'
    $withOneLiner = if ($originalPath) { $originalPath.TrimEnd(';') + ';' + $oneLinerDir } else { $oneLinerDir }
    Set-UserPath $withOneLiner
    Invoke-Setup 'over-one-liner'
    Test-Installed
    Assert ((Get-UserPath) -ceq $withOneLiner) 'Setup left the PATH entry the install line made as it was'
    Invoke-Setup 'reinstall'
    Test-Installed
    New-DebugLog
    Invoke-Uninstall 'uninstall-over-one-liner'
    Test-Removed
    Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'Uninstall took the install line''s PATH entry off too, and only that'
} catch {
    foreach ($log in Get-ChildItem -LiteralPath $LogDir -Filter '*.log' -ErrorAction SilentlyContinue) {
        Write-Host "---- $($log.Name)"
        Get-Content -LiteralPath $log.FullName | Write-Host
    }
    throw
} finally {
    Set-UserPath $originalPath
}
Write-Host 'The installer installs, upgrades and uninstalls cleanly.'

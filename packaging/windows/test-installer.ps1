# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
#
# Installs wopr_<version>_windows_setup.exe silently for the current user, checks what it installed,
# uninstalls it and checks that it left nothing behind, not even a debug log. Each installed file
# must be its release file, byte for byte, as checksums.txt in -BinDir lists it, and the folder holds
# nothing else. Then it does the same over a copy that the PowerShell install line put in
# %LOCALAPPDATA%\Programs\wopr, the same folder, and installs twice. Then it checks whose PATH entry
# Uninstall removes: one that Setup added, also after upgrades, but not one that was already there
# for a folder that /DIR= named. It checks that Setup leaves a Path that is not text alone, and adds
# no folder whose name has a semicolon or a percent sign, and that Uninstall notices a wopr.exe in
# use. With -AllUsers, as an administrator, it last installs for all users (/ALLUSERS) and checks
# the system PATH.
# Run it on a machine without WOPR installed (a CI runner): it changes the user's PATH (and with
# -AllUsers the system's) while it runs, and puts it back at the end.
#
#   pwsh -File packaging/windows/test-installer.ps1 -Installer build/installer/wopr_1.2.3_windows_setup.exe -Version 1.2.3 -BinDir dist/release
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Installer,
    [Parameter(Mandatory)] [string] $Version,
    # The release files that the installer was built from, with their checksums.txt.
    [Parameter(Mandatory)] [string] $BinDir,
    # Where Setup's and Uninstall's logs go; printed when a check fails.
    [string] $LogDir = (Join-Path ([IO.Path]::GetTempPath()) 'wopr-installer-logs'),
    # Also test an install for all users. It needs an administrator.
    [switch] $AllUsers
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$Installer = (Resolve-Path -LiteralPath $Installer).Path
$BinDir = (Resolve-Path -LiteralPath $BinDir).Path
$null = New-Item -ItemType Directory -Force -Path $LogDir
$app = Join-Path $env:LOCALAPPDATA 'Programs\WOPR'
$oneLinerDir = Join-Path $env:LOCALAPPDATA 'Programs\wopr' # the PowerShell install line's folder
# Folders for /DIR=: one that other programs share and that PATH already has, one that is new, and
# one whose name has a semicolon.
$sharedDir = Join-Path $env:LOCALAPPDATA 'wopr-test-shared'
$ownDir = Join-Path $env:LOCALAPPDATA 'wopr-test-own'
$semicolonDir = Join-Path $env:LOCALAPPDATA 'wopr-test;semicolon'
$percentDir = Join-Path $env:LOCALAPPDATA 'wopr-test%percent'
# An install for all users: Program Files, the machine's uninstall key, Start menu and PATH.
$commonApp = Join-Path $env:ProgramFiles 'WOPR'
$machineUninstallKey = 'HKLM:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{A7D86110-58E9-404E-9164-FBDDBDC6AC22}_is1'
$commonShortcut = Join-Path ([Environment]::GetFolderPath('CommonPrograms')) 'WOPR.lnk'
$machineEnvironment = 'SYSTEM\CurrentControlSet\Control\Session Manager\Environment'
$programs = [Environment]::GetFolderPath('Programs') # the user's Start menu, All apps
$shortcut = Join-Path $programs 'WOPR.lnk'
$desktopShortcut = Join-Path ([Environment]::GetFolderPath('Desktop')) 'WOPR.lnk'
$debugLogDir = Join-Path $env:LOCALAPPDATA 'wopr' # internal/debuglog's folder: os.UserCacheDir()\wopr
# wopr.iss's AppId, plus the _is1 that Inno Setup adds.
$uninstallKey = 'HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{A7D86110-58E9-404E-9164-FBDDBDC6AC22}_is1'
# wopr.iss's record of the folder that Setup put on PATH (SetPreviousData).
$addedToPathValue = 'Inno Setup CodeFile: AddedToPath'
$files = 'wopr.exe', 'LICENSE.txt', 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md', 'unins000.exe', 'unins000.dat'
$wantMachine, $goarch = switch ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64' { 0x8664, 'amd64' }
    'Arm64' { 0xAA64, 'arm64' }
    default { throw "no Windows build for $_" }
}
# Each installed file that is a release file, and the release file's name in checksums.txt.
$releaseNames = [ordered]@{
    'wopr.exe' = "wopr_${Version}_windows_$goarch.exe"
    'LICENSE.txt' = 'LICENSE'
    'NOTICE.md' = 'NOTICE.md'
    'THIRD_PARTY_NOTICES.txt' = 'THIRD_PARTY_NOTICES.txt'
    'README.md' = 'README.md'
}
$sums = @{}
foreach ($line in Get-Content -LiteralPath (Join-Path $BinDir 'checksums.txt')) {
    if ($line -match '^([0-9a-f]{64})  (\S+)$') {
        $sums[$Matches[2]] = $Matches[1]
    }
}
foreach ($name in $releaseNames.Values) {
    if (-not $sums.ContainsKey($name)) { throw "checksums.txt in $BinDir does not list $name" }
}

# The user's PATH as stored: unexpanded, or $null when there is none. Not only a string: a test
# below stores a REG_MULTI_SZ, which comes back as an array.
function Get-UserPath {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
    try {
        if ($null -eq $key) { return $null }
        return $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    } finally {
        if ($null -ne $key) { $key.Dispose() }
    }
}

function Get-UserPathKind {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey('Environment')
    try {
        if ($null -eq $key -or $null -eq $key.GetValue('Path', $null)) { return $null }
        return $key.GetValueKind('Path')
    } finally {
        if ($null -ne $key) { $key.Dispose() }
    }
}

# Untyped, so that $null (no Path value at all) stays $null rather than becoming "".
function Set-UserPath($Value, [Microsoft.Win32.RegistryValueKind] $Kind = 'ExpandString') {
    $key = [Microsoft.Win32.Registry]::CurrentUser.CreateSubKey('Environment')
    try {
        if ($null -eq $Value) {
            $key.DeleteValue('Path', $false)
        } else {
            $key.SetValue('Path', $Value, $Kind)
        }
    } finally {
        $key.Dispose()
    }
}

# The PATH's entries, without empty ones (Setup may drop a trailing ";").
function Get-PathEntries([AllowNull()] [string] $Path) {
    return @("$Path" -split ';' | Where-Object { $_ -ne '' })
}

# How many PATH entries name $Dir, ignoring case and a trailing backslash, as Windows does.
function Get-PathCount([string] $Dir = $app) {
    return @(Get-PathEntries (Get-UserPath) | Where-Object { $_.Trim().TrimEnd('\') -eq $Dir }).Count
}

# A value of the uninstall entry ($Key: the user's, or the machine's), or $null.
function Get-UninstallValue([string] $Name, [string] $Key = $uninstallKey) {
    if (-not (Test-Path -LiteralPath $Key)) { return $null }
    $value = (Get-ItemProperty -LiteralPath $Key).PSObject.Properties[$Name]
    if ($null -eq $value) { return $null }
    return $value.Value
}

# The folder that wopr.iss recorded as put on PATH by Setup, or $null.
function Get-AddedToPath([string] $Key = $uninstallKey) {
    return Get-UninstallValue $addedToPathValue $Key
}

# The machine's PATH as stored, unexpanded, or $null.
function Get-SystemPath {
    $key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey($machineEnvironment)
    try {
        return $key.GetValue('Path', $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    } finally {
        $key.Dispose()
    }
}

function Set-SystemPath([string] $Value) {
    $key = [Microsoft.Win32.Registry]::LocalMachine.OpenSubKey($machineEnvironment, $true)
    try {
        $key.SetValue('Path', $Value, [Microsoft.Win32.RegistryValueKind]::ExpandString)
    } finally {
        $key.Dispose()
    }
}

function Assert([bool] $Condition, [string] $Message) {
    if (-not $Condition) { throw "FAILED: $Message" }
    Write-Host "ok: $Message"
}

# For the current user, or with -ForAll for all users (/ALLUSERS).
function Invoke-Setup([string] $Name, [string[]] $Extra = @(), [switch] $ForAll) {
    $log = Join-Path $LogDir "$Name.log"
    $mode = if ($ForAll) { '/ALLUSERS' } else { '/CURRENTUSER' }
    $p = Start-Process -FilePath $Installer -Wait -PassThru -ArgumentList (@(
            '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', $mode, "/LOG=`"$log`"") + $Extra)
    Assert ($p.ExitCode -eq 0) "Setup ($Name) exits with 0, not $($p.ExitCode)"
}

# With -Kept, $Dir holds other files, so Uninstall leaves the folder.
function Invoke-Uninstall([string] $Name, [string] $Dir = $app, [switch] $Kept) {
    $log = Join-Path $LogDir "$Name.log"
    $uninstaller = Join-Path $Dir 'unins000.exe'
    $p = Start-Process -FilePath $uninstaller -Wait -PassThru -ArgumentList @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', "/LOG=`"$log`"")
    Assert ($p.ExitCode -eq 0) "Uninstall ($Name) exits with 0, not $($p.ExitCode)"
    # The uninstaller runs a copy of itself from TEMP, which can still be removing the folder, or
    # retrying the Start menu shortcut (wopr.iss, DeleteStartMenuShortcut), when the first one exits.
    $leftover = if ($Kept) { $uninstaller } else { $Dir }
    for ($i = 0; $i -lt 60 -and ((Test-Path -LiteralPath $leftover) -or (Test-Path -LiteralPath $uninstallKey) -or (Test-Path -LiteralPath $shortcut)); $i++) {
        Start-Sleep -Seconds 1
    }
}

# What Setup installed in $Dir. With -Only, the folder holds nothing else.
function Test-Installed([string] $Dir = $app, [switch] $Only) {
    foreach ($f in $files) {
        Assert (Test-Path -LiteralPath (Join-Path $Dir $f) -PathType Leaf) "$f is in $Dir"
    }
    if ($Only) {
        $have = (@(Get-ChildItem -LiteralPath $Dir -Force -Name) | Sort-Object) -join ', '
        $want = (@($files) | Sort-Object) -join ', '
        Assert ($have -eq $want) "$Dir holds only what Setup installs: $have"
    }
    # The release files, unchanged: the installer holds what the build made.
    foreach ($installed in $releaseNames.Keys) {
        $release = $releaseNames[$installed]
        $hash = (Get-FileHash -LiteralPath (Join-Path $Dir $installed) -Algorithm SHA256).Hash.ToLowerInvariant()
        Assert ($hash -ceq $sums[$release]) "$installed is $release, as checksums.txt lists it: $hash"
    }
    $exe = Join-Path $Dir 'wopr.exe'

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
    Assert ($workingDir -eq $Dir) "the shortcut starts in ${Dir}: $workingDir"
    Assert (-not (Test-Path -LiteralPath (Join-Path $programs 'WOPR') -PathType Container)) 'there is no WOPR folder in the Start menu'
    Assert (-not (Test-Path -LiteralPath $desktopShortcut)) 'there is no desktop shortcut (that task is off by default)'

    Assert (Test-Path -LiteralPath $uninstallKey) "Settings > Apps lists WOPR ($uninstallKey)"
    $entry = Get-ItemProperty -LiteralPath $uninstallKey
    Assert ($entry.DisplayName -eq 'WOPR') "its name is WOPR: $($entry.DisplayName)"
    Assert ($entry.DisplayVersion -eq $Version) "its version is ${Version}: $($entry.DisplayVersion)"
    Assert ($entry.Publisher -eq 'Christopher Goes') "its publisher is Christopher Goes: $($entry.Publisher)"
    Assert ($entry.DisplayIcon -eq $exe) "its icon is wopr.exe's: $($entry.DisplayIcon)"

    Assert ((Get-PathCount $Dir) -eq 1) "PATH has $Dir once: $(Get-UserPath)"
    Assert ((Get-UserPathKind) -eq 'ExpandString') "PATH is still REG_EXPAND_SZ: $(Get-UserPathKind)"
}

# What Uninstall left. With -Kept, $Dir was there before Setup and holds other files, so it stays.
function Test-Removed([string] $Dir = $app, [switch] $Kept) {
    foreach ($f in $files) {
        Assert (-not (Test-Path -LiteralPath (Join-Path $Dir $f))) "$f is gone from $Dir"
    }
    if ($Kept) {
        Assert (Test-Path -LiteralPath $Dir -PathType Container) "$Dir is still there"
    } else {
        Assert (-not (Test-Path -LiteralPath $Dir)) "$Dir is gone"
    }
    Assert (-not (Test-Path -LiteralPath $shortcut)) 'the Start menu shortcut is gone'
    Assert (-not (Test-Path -LiteralPath $uninstallKey)) 'Settings > Apps no longer lists WOPR'
    Assert (-not (Test-Path -LiteralPath $debugLogDir)) "the debug log and its folder, $debugLogDir, are gone"
}

# A debug log, as WOPR_DEBUG=1 writes it, for Uninstall to delete.
function New-DebugLog {
    $null = New-Item -ItemType Directory -Force -Path $debugLogDir
    Set-Content -LiteralPath (Join-Path $debugLogDir 'debug.log') -Value 'a debug log'
}

# The original PATH with $Extra appended, as one string.
function Join-UserPath($Original, [string] $Extra) {
    if ($Original) { return $Original.TrimEnd(';') + ';' + $Extra }
    return $Extra
}

$originalPath = Get-UserPath
$originalKind = Get-UserPathKind
Assert ($null -eq $originalPath -or $originalPath -is [string]) "the user's PATH is a string or absent: $originalKind"
Assert (-not (Test-Path -LiteralPath $app) -and -not (Test-Path -LiteralPath $uninstallKey) -and (Get-PathCount) -eq 0) 'WOPR is not installed yet'
foreach ($dir in $sharedDir, $ownDir, $semicolonDir, $percentDir) {
    Assert (-not (Test-Path -LiteralPath $dir)) "$dir does not exist yet"
}
$script:systemPathToRestore = $null
try {
    Write-Host "== A first install, with the default tasks"
    Invoke-Setup 'install'
    Test-Installed -Only
    Assert ((Get-AddedToPath) -eq $app) "Setup recorded that it added $app to PATH: $(Get-AddedToPath)"
    New-DebugLog
    Invoke-Uninstall 'uninstall'
    Test-Removed
    Assert ((Get-PathCount) -eq 0) "PATH no longer has ${app}: $(Get-UserPath)"
    Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'PATH is back to what it was, every other entry untouched'

    Write-Host "== Over the PowerShell install line's copy, then again over itself"
    $null = New-Item -ItemType Directory -Force -Path $oneLinerDir
    Set-Content -LiteralPath (Join-Path $oneLinerDir 'wopr.exe') -Value 'an older wopr.exe'
    $withOneLiner = Join-UserPath $originalPath $oneLinerDir
    Set-UserPath $withOneLiner
    Invoke-Setup 'over-one-liner'
    Test-Installed -Only
    Assert ((Get-UserPath) -ceq $withOneLiner) 'Setup left the PATH entry the install line made as it was'
    Assert ($null -eq (Get-AddedToPath)) 'Setup recorded no PATH entry of its own'
    Invoke-Setup 'reinstall'
    Test-Installed -Only
    New-DebugLog
    Invoke-Uninstall 'uninstall-over-one-liner'
    Test-Removed
    Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'Uninstall took the install line''s PATH entry off too (the default folder), and only that'

    Write-Host "== Into a folder that other programs share and that PATH already has (/DIR=)"
    $null = New-Item -ItemType Directory -Path $sharedDir
    Set-Content -LiteralPath (Join-Path $sharedDir 'other-tool.txt') -Value 'another program''s file'
    $withShared = Join-UserPath $originalPath $sharedDir
    Set-UserPath $withShared
    Invoke-Setup 'shared' @("/DIR=`"$sharedDir`"")
    Test-Installed $sharedDir
    Assert ((Get-UserPath) -ceq $withShared) 'Setup left PATH as it was'
    Assert ($null -eq (Get-AddedToPath)) 'Setup recorded no PATH entry of its own'
    Invoke-Uninstall 'uninstall-shared' $sharedDir -Kept
    Test-Removed $sharedDir -Kept
    Assert (Test-Path -LiteralPath (Join-Path $sharedDir 'other-tool.txt')) 'the other program''s file is still there'
    Assert ((Get-UserPath) -ceq $withShared) 'Uninstall kept the shared folder on PATH, which Setup did not add'
    Remove-Item -LiteralPath $sharedDir -Recurse -Force
    Set-UserPath $originalPath $(if ($null -ne $originalKind) { $originalKind } else { 'ExpandString' })

    Write-Host "== Into a new folder (/DIR=), then upgrades: with the same folder, and with the PATH task off"
    Invoke-Setup 'own' @("/DIR=`"$ownDir`"")
    Test-Installed $ownDir -Only
    Assert ((Get-AddedToPath) -eq $ownDir) "Setup recorded that it added ${ownDir}: $(Get-AddedToPath)"
    Invoke-Setup 'own-upgrade' # Setup uses the installed folder again
    Test-Installed $ownDir -Only
    Assert ((Get-AddedToPath) -eq $ownDir) "the upgrade kept the record: $(Get-AddedToPath)"
    Invoke-Setup 'own-upgrade-no-path' @('/MERGETASKS="!path"')
    $deselected = @("$(Get-UninstallValue 'Inno Setup: Deselected Tasks')" -split ',')
    Assert ($deselected -contains 'path') "the upgrade ran with the PATH task off: deselected $($deselected -join ',')"
    Assert ((Get-PathCount $ownDir) -eq 1) "PATH still has $ownDir once"
    Assert ((Get-AddedToPath) -eq $ownDir) "an upgrade with the PATH task off kept the record too: $(Get-AddedToPath)"
    New-DebugLog
    Invoke-Uninstall 'uninstall-own' $ownDir
    Test-Removed $ownDir
    Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'Uninstall took the folder that Setup added off PATH, and only that'

    Write-Host "== A Path that is not text (REG_MULTI_SZ)"
    $multi = [string[]]@('C:\wopr-test\one', 'C:\wopr-test\two')
    Set-UserPath $multi 'MultiString'
    Invoke-Setup 'multi-sz'
    Assert ((Get-UserPathKind) -eq 'MultiString') "Setup left Path's kind as it was: $(Get-UserPathKind)"
    Assert (((Get-UserPath) -join '|') -ceq ($multi -join '|')) "Setup left Path's value as it was: $((Get-UserPath) -join '|')"
    Assert (Select-String -LiteralPath (Join-Path $LogDir 'multi-sz.log') -SimpleMatch -Quiet 'PATH is not a text value') 'Setup said why it did not add WOPR to PATH'
    Invoke-Uninstall 'uninstall-multi-sz'
    Assert (((Get-UserPath) -join '|') -ceq ($multi -join '|')) 'Uninstall left Path alone too'
    Test-Removed
    Set-UserPath $originalPath $(if ($null -ne $originalKind) { $originalKind } else { 'ExpandString' })

    foreach ($badDir in $semicolonDir, $percentDir) {
        Write-Host "== A folder whose name PATH cannot hold (/DIR=$badDir)"
        $name = Split-Path -Leaf $badDir
        Invoke-Setup $name @("/DIR=`"$badDir`"")
        Assert ((Test-Path -LiteralPath (Join-Path $badDir 'wopr.exe') -PathType Leaf)) "Setup installed into $badDir"
        Assert (((Get-PathEntries (Get-UserPath)) -join ';') -ceq ((Get-PathEntries $originalPath) -join ';')) 'Setup left PATH as it was'
        Assert ($null -eq (Get-AddedToPath)) 'Setup recorded no PATH entry'
        Assert (Select-String -LiteralPath (Join-Path $LogDir "$name.log") -SimpleMatch -Quiet 'semicolon or a percent sign') 'Setup said why it did not add WOPR to PATH'
        Invoke-Uninstall "uninstall-$name" $badDir
        Test-Removed $badDir
    }

    Write-Host "== Uninstall while wopr.exe is in use"
    Invoke-Setup 'in-use'
    $exe = Join-Path $app 'wopr.exe'
    # Opened with no sharing, as Windows holds a running program: Uninstall cannot open it to write.
    $held = [IO.File]::Open($exe, [IO.FileMode]::Open, [IO.FileAccess]::Read, [IO.FileShare]::None)
    try {
        Invoke-Uninstall 'uninstall-in-use' -Kept
        Assert (Select-String -LiteralPath (Join-Path $LogDir 'uninstall-in-use.log') -SimpleMatch -Quiet 'wopr.exe is in use') 'a silent Uninstall noticed that wopr.exe is in use, and said so'
        Assert (Test-Path -LiteralPath $exe -PathType Leaf) 'and could not remove it'
    } finally {
        $held.Dispose()
    }
    Remove-Item -LiteralPath $app -Recurse -Force
    Assert (-not (Test-Path -LiteralPath $uninstallKey)) 'Uninstall removed the rest'

    if ($AllUsers) {
        Write-Host "== For all users (/ALLUSERS)"
        $admin = [Security.Principal.WindowsPrincipal]::new([Security.Principal.WindowsIdentity]::GetCurrent())
        Assert ($admin.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) 'this test runs as an administrator'
        Assert (-not (Test-Path -LiteralPath $commonApp) -and -not (Test-Path -LiteralPath $machineUninstallKey)) 'WOPR is not installed for all users yet'
        $originalSystemPath = Get-SystemPath
        Assert ($originalSystemPath -is [string]) 'the system PATH is a string'
        $script:systemPathToRestore = $originalSystemPath
        $userPathBefore = Get-UserPath
        Invoke-Setup 'all-users' -ForAll
        foreach ($installed in $releaseNames.Keys) {
            $hash = (Get-FileHash -LiteralPath (Join-Path $commonApp $installed) -Algorithm SHA256).Hash.ToLowerInvariant()
            Assert ($hash -ceq $sums[$releaseNames[$installed]]) "$installed in $commonApp is $($releaseNames[$installed])"
        }
        Assert (Test-Path -LiteralPath $commonShortcut -PathType Leaf) "the Start menu shortcut for all users is $commonShortcut"
        Assert (Test-Path -LiteralPath $machineUninstallKey) "Settings > Apps lists WOPR for all users ($machineUninstallKey)"
        Assert ((Get-SystemPath) -ceq ($originalSystemPath.TrimEnd(';') + ';' + $commonApp)) "the system PATH gained $commonApp, every other entry as it was"
        Assert ((Get-AddedToPath $machineUninstallKey) -eq $commonApp) "Setup recorded that it added ${commonApp}: $(Get-AddedToPath $machineUninstallKey)"
        Assert ((Get-UserPath) -ceq $userPathBefore) "the user's PATH is unchanged"
        Invoke-Uninstall 'uninstall-all-users' $commonApp
        Assert (-not (Test-Path -LiteralPath $commonApp)) "$commonApp is gone"
        Assert (-not (Test-Path -LiteralPath $machineUninstallKey)) 'Settings > Apps no longer lists WOPR for all users'
        Assert (-not (Test-Path -LiteralPath $commonShortcut)) 'the Start menu shortcut for all users is gone'
        Assert ((Get-SystemPath) -ceq $originalSystemPath) 'Uninstall put the system PATH back as it was'
    }
} catch {
    foreach ($log in Get-ChildItem -LiteralPath $LogDir -Filter '*.log' -ErrorAction SilentlyContinue) {
        Write-Host "---- $($log.Name)"
        Get-Content -LiteralPath $log.FullName | Write-Host
    }
    throw
} finally {
    # Uninstall whatever a failed check left installed, so that later steps start clean.
    foreach ($dir in $app, $sharedDir, $ownDir, $semicolonDir, $percentDir, $commonApp) {
        $uninstaller = Join-Path $dir 'unins000.exe'
        if (Test-Path -LiteralPath $uninstaller) {
            Start-Process -FilePath $uninstaller -Wait -ArgumentList '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART' -ErrorAction SilentlyContinue
        }
    }
    Set-UserPath $originalPath $(if ($null -ne $originalKind) { $originalKind } else { 'ExpandString' })
    if ($null -ne $script:systemPathToRestore) { Set-SystemPath $script:systemPathToRestore }
    foreach ($dir in $sharedDir, $ownDir, $semicolonDir, $percentDir) {
        Remove-Item -LiteralPath $dir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
Write-Host 'The installer installs, upgrades and uninstalls cleanly, and removes only the PATH entry it added.'

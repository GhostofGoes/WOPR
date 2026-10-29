# SPDX-FileCopyrightText: 2026 Christopher Goes
# SPDX-License-Identifier: MIT
#
# Builds wopr_<version>_windows_setup.exe with Inno Setup from the release files: the two Windows
# programs and the notices, already checked against checksums.txt by the caller.
#
#   pwsh -File packaging/windows/build-installer.ps1 -Version 1.2.3 -BinDir dist/release -OutputDir build/installer
#
# With -ExpectSame, it then checks that the installer it built is the given one, byte for byte. The
# build is reproducible: Inno Setup writes no time or path into it (every [Files] entry in wopr.iss
# is notimestamp), so the same release files, wopr.iss and Inno Setup give the same bytes on any
# computer. windows.yml's smoke jobs rebuild the installer this way on other runners.
#
# Inno Setup comes from its immutable GitHub release, pinned by SHA-256 (the .issig signature file
# beside it and jrsoftware.org's copy of that file say the same) and checked for its publisher's
# Authenticode signature before it runs. It is installed into a new folder of its own in portable
# mode: no uninstall entry, no Start menu entries, no file associations, and no administrator rights.
# Each of its programs and DLLs must then carry the same publisher's valid signature; ISCC checks the
# Setup stubs (.e32, .e64) against their .issig files itself. The download and, unless -InnoDir names
# a folder, the installed copy are in a new temporary folder that the script removes at the end.
# Its licence lets anyone use it; jrsoftware.org asks only commercial users (more than USD 5,000 a
# year in revenue, donations included) to buy a licence (https://jrsoftware.org/isorder.php).
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Version,
    [Parameter(Mandatory)] [string] $BinDir,
    [Parameter(Mandatory)] [string] $OutputDir,
    # Where Inno Setup is installed: a folder that does not exist yet, or is empty. The default is a
    # new temporary folder, removed at the end.
    [string] $InnoDir,
    # An installer that the one built here must equal, byte for byte.
    [string] $ExpectSame
)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue' # the progress bar slows Invoke-WebRequest down
Set-StrictMode -Version Latest

# Bump all three together, from https://jrsoftware.org/isdl.php and the release's .issig file.
$InnoUrl = 'https://github.com/jrsoftware/issrc/releases/download/is-7_1_0/innosetup-7.1.0-x64.exe'
$InnoSha256 = '0362A383ED217D4C4239B5933866DD96D3EB2102737DA92F80F6057A4B40DF2F'
$InnoSigner = 'Pyrsys B.V.'

if ($Version -notmatch '^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$') {
    throw "Version '$Version' is not X.Y.Z or X.Y.Z-suffix"
}
$BinDir = (Resolve-Path -LiteralPath $BinDir).Path
if ($ExpectSame) {
    $ExpectSame = (Resolve-Path -LiteralPath $ExpectSame).Path
}
$null = New-Item -ItemType Directory -Force -Path $OutputDir
$OutputDir = (Resolve-Path -LiteralPath $OutputDir).Path
$issFile = Join-Path $PSScriptRoot 'wopr.iss'
$installer = Join-Path $OutputDir "wopr_${Version}_windows_setup.exe"
if ($ExpectSame -and $ExpectSame -eq $installer) {
    throw "-ExpectSame names the file that this build writes; give the rebuild another -OutputDir"
}

foreach ($name in "wopr_${Version}_windows_amd64.exe", "wopr_${Version}_windows_arm64.exe",
    'LICENSE', 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md') {
    if (-not (Test-Path -LiteralPath (Join-Path $BinDir $name) -PathType Leaf)) {
        throw "$name is missing from $BinDir"
    }
}

# The Authenticode signer's name, after checking that the signature is valid.
function Get-ValidSigner([string] $Path) {
    $sig = Get-AuthenticodeSignature -LiteralPath $Path
    if ($sig.Status -ne 'Valid') {
        throw "$Path`: its Authenticode signature is $($sig.Status): $($sig.StatusMessage)"
    }
    return $sig.SignerCertificate.GetNameInfo([Security.Cryptography.X509Certificates.X509NameType]::SimpleName, $false)
}

$work = Join-Path ([IO.Path]::GetTempPath()) "wopr-innosetup-$([guid]::NewGuid())"
$null = New-Item -ItemType Directory -Path $work
try {
    if (-not $InnoDir) {
        $InnoDir = Join-Path $work 'innosetup'
    } elseif ((Test-Path -LiteralPath $InnoDir) -and @(Get-ChildItem -LiteralPath $InnoDir -Force).Count -gt 0) {
        throw "-InnoDir $InnoDir is not empty; give a new folder, so that no file from before is used"
    }

    $download = Join-Path $work (Split-Path -Leaf ([uri]$InnoUrl).AbsolutePath)
    Write-Host "Downloading $InnoUrl"
    Invoke-WebRequest -Uri $InnoUrl -OutFile $download -MaximumRetryCount 3 -RetryIntervalSec 5
    $hash = (Get-FileHash -LiteralPath $download -Algorithm SHA256).Hash
    if ($hash -ne $InnoSha256) {
        throw "Inno Setup's SHA-256 is $hash, not $InnoSha256"
    }
    $signer = Get-ValidSigner $download
    if ($signer -ne $InnoSigner) {
        throw "Inno Setup is signed by '$signer', not '$InnoSigner'"
    }
    Write-Host "Inno Setup: SHA-256 $hash, signed by $signer"

    # /PORTABLE=1 is Inno Setup's own installer's portable mode; /CURRENTUSER keeps it from asking
    # for administrator rights.
    $iscc = Join-Path $InnoDir 'ISCC.exe'
    $log = Join-Path $work 'innosetup-install.log'
    $setup = Start-Process -FilePath $download -Wait -PassThru -ArgumentList @(
        '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', '/CURRENTUSER', '/PORTABLE=1', '/NOICONS',
        "/DIR=`"$InnoDir`"", "/LOG=`"$log`"")
    if ($setup.ExitCode -ne 0 -or -not (Test-Path -LiteralPath $iscc -PathType Leaf)) {
        if (Test-Path -LiteralPath $log) { Get-Content -LiteralPath $log | Write-Host }
        throw "Installing Inno Setup failed with exit code $($setup.ExitCode)"
    }
    # Every program and DLL that the compiler may load, ISCC.exe and ISCmplr.dll first among them.
    $binaries = @(Get-ChildItem -LiteralPath $InnoDir -Recurse -File | Where-Object Extension -In '.exe', '.dll')
    foreach ($file in $binaries) {
        $fileSigner = Get-ValidSigner $file.FullName
        if ($fileSigner -ne $InnoSigner) {
            throw "$($file.FullName) is signed by '$fileSigner', not '$InnoSigner'"
        }
    }
    Write-Host "ISCC $((Get-Item -LiteralPath $iscc).VersionInfo.ProductVersion) at $iscc; $($binaries.Count) programs and DLLs signed by $InnoSigner"

    & $iscc '/Qp' "/DVersion=$Version" "/DBinDir=$BinDir" "/O$OutputDir" $issFile
    if ($LASTEXITCODE -ne 0) {
        throw "ISCC failed with exit code $LASTEXITCODE"
    }
} finally {
    Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
}

if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) {
    throw "ISCC did not write $installer"
}
$item = Get-Item -LiteralPath $installer
$sum = (Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash
Write-Host "Built $($item.FullName): $($item.Length) bytes, SHA-256 $sum"

if ($ExpectSame) {
    $want = (Get-FileHash -LiteralPath $ExpectSame -Algorithm SHA256).Hash
    if ($sum -ne $want) {
        throw "the installer built here (SHA-256 $sum) is not $ExpectSame (SHA-256 $want)"
    }
    Write-Host "Same as ${ExpectSame}: SHA-256 $want"
}

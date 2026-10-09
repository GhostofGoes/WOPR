# Builds wopr_<version>_windows_setup.exe with Inno Setup from the release files: the two Windows
# programs and the notices, already checked against checksums.txt by the caller.
#
#   pwsh -File packaging/windows/build-installer.ps1 -Version 1.2.3 -BinDir dist/release -OutputDir build/installer
#
# Inno Setup comes from its immutable GitHub release, pinned by SHA-256 (the .issig signature file
# beside it and jrsoftware.org's copy of that file say the same) and checked for its publisher's
# Authenticode signature before it runs. It is installed into a folder of its own in portable mode:
# no uninstall entry, no Start menu entries, no file associations, and no administrator rights.
# Its licence lets anyone use it; jrsoftware.org asks only commercial users (more than USD 5,000 a
# year in revenue, donations included) to buy a licence (https://jrsoftware.org/isorder.php).
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Version,
    [Parameter(Mandatory)] [string] $BinDir,
    [Parameter(Mandatory)] [string] $OutputDir,
    # Where Inno Setup is installed (over any copy already there).
    [string] $InnoDir = (Join-Path ([IO.Path]::GetTempPath()) 'wopr-innosetup-7.1.0')
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
$null = New-Item -ItemType Directory -Force -Path $OutputDir
$OutputDir = (Resolve-Path -LiteralPath $OutputDir).Path
$issFile = Join-Path $PSScriptRoot 'wopr.iss'

foreach ($name in "wopr_${Version}_windows_amd64.exe", "wopr_${Version}_windows_arm64.exe",
    'LICENSE', 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md') {
    if (-not (Test-Path -LiteralPath (Join-Path $BinDir $name) -PathType Leaf)) {
        throw "$name is missing from $BinDir"
    }
}

$download = Join-Path ([IO.Path]::GetTempPath()) 'innosetup-7.1.0-x64.exe'
Write-Host "Downloading $InnoUrl"
Invoke-WebRequest -Uri $InnoUrl -OutFile $download -MaximumRetryCount 3 -RetryIntervalSec 5
$hash = (Get-FileHash -LiteralPath $download -Algorithm SHA256).Hash
if ($hash -ne $InnoSha256) {
    throw "Inno Setup's SHA-256 is $hash, not $InnoSha256"
}
$sig = Get-AuthenticodeSignature -LiteralPath $download
if ($sig.Status -ne 'Valid') {
    throw "Inno Setup's Authenticode signature is $($sig.Status): $($sig.StatusMessage)"
}
$signer = $sig.SignerCertificate.GetNameInfo([Security.Cryptography.X509Certificates.X509NameType]::SimpleName, $false)
if ($signer -ne $InnoSigner) {
    throw "Inno Setup is signed by '$signer' ($($sig.SignerCertificate.Subject)), not '$InnoSigner'"
}
Write-Host "Inno Setup: SHA-256 $hash, signed by $($sig.SignerCertificate.Subject)"

# /PORTABLE=1 is Inno Setup's own installer's portable mode; /CURRENTUSER keeps it from asking for
# administrator rights.
$iscc = Join-Path $InnoDir 'ISCC.exe'
$log = Join-Path ([IO.Path]::GetTempPath()) 'innosetup-install.log'
$setup = Start-Process -FilePath $download -Wait -PassThru -ArgumentList @(
    '/VERYSILENT', '/SUPPRESSMSGBOXES', '/NORESTART', '/SP-', '/CURRENTUSER', '/PORTABLE=1', '/NOICONS',
    "/DIR=`"$InnoDir`"", "/LOG=`"$log`"")
if ($setup.ExitCode -ne 0 -or -not (Test-Path -LiteralPath $iscc -PathType Leaf)) {
    if (Test-Path -LiteralPath $log) { Get-Content -LiteralPath $log | Write-Host }
    throw "Installing Inno Setup failed with exit code $($setup.ExitCode)"
}
Write-Host "ISCC $((Get-Item -LiteralPath $iscc).VersionInfo.ProductVersion) at $iscc"

& $iscc '/Qp' "/DVersion=$Version" "/DBinDir=$BinDir" "/O$OutputDir" $issFile
if ($LASTEXITCODE -ne 0) {
    throw "ISCC failed with exit code $LASTEXITCODE"
}

$installer = Join-Path $OutputDir "wopr_${Version}_windows_setup.exe"
if (-not (Test-Path -LiteralPath $installer -PathType Leaf)) {
    throw "ISCC did not write $installer"
}
$item = Get-Item -LiteralPath $installer
Write-Host "Built $($item.FullName): $($item.Length) bytes, SHA-256 $((Get-FileHash -LiteralPath $installer -Algorithm SHA256).Hash)"

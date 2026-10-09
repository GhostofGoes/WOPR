# Checks the release files that the Windows installer and the MSIX packages are built from against
# checksums.txt beside them: the two Windows programs and the notices. Those are the attested files
# of the release, so the packages hold nothing that the build did not make.
#
#   pwsh -File packaging/windows/check-release-files.ps1 -Version 1.2.3 -BinDir dist/release
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Version,
    [Parameter(Mandatory)] [string] $BinDir
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$sums = @{}
foreach ($line in Get-Content -LiteralPath (Join-Path $BinDir 'checksums.txt')) {
    if ($line -match '^([0-9a-f]{64})  (\S+)$') {
        $sums[$Matches[2]] = $Matches[1]
    }
}
foreach ($name in "wopr_${Version}_windows_amd64.exe", "wopr_${Version}_windows_arm64.exe",
    'LICENSE', 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md') {
    if (-not $sums.ContainsKey($name)) {
        throw "checksums.txt does not list $name"
    }
    $hash = (Get-FileHash -LiteralPath (Join-Path $BinDir $name) -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($hash -cne $sums[$name]) {
        throw "$name's SHA-256 is $hash, but checksums.txt says $($sums[$name])"
    }
    Write-Host "ok: $name $hash"
}

# Builds the MSIX packages for a later Microsoft Store submission from the release files: one
# package per architecture, wopr_<version>_windows_amd64.msix and wopr_<version>_windows_arm64.msix,
# and the bundle that holds both, wopr_<version>_windows.msixbundle, which is what Partner Center
# takes. They are unsigned: the Store signs what it publishes, and nobody can install an unsigned
# package, so they are workflow artifacts and never release files.
#
#   pwsh -File packaging/windows/build-msix.ps1 -Version 1.2.3 -BinDir dist/release -OutputDir build/msix
#
# The Store's identity comes from -IdentityName, -Publisher and -PublisherDisplayName (the workflow
# passes the repository variables MSIX_IDENTITY_NAME, MSIX_PUBLISHER and MSIX_PUBLISHER_DISPLAY_NAME);
# until the owner reserves the name, they are placeholders. The package version is X.Y.Z.0, without
# a snapshot suffix. The Store refuses a version whose first number is 0, so a 0.x build packs but
# cannot be submitted.
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $Version,
    [Parameter(Mandatory)] [string] $BinDir,
    [Parameter(Mandatory)] [string] $OutputDir,
    [string] $IdentityName,
    [string] $Publisher,
    [string] $PublisherDisplayName,
    # Scratch space for the package folders; a fresh folder under the temporary directory by default.
    [string] $WorkDir = (Join-Path ([IO.Path]::GetTempPath()) "wopr-msix-$([guid]::NewGuid())")
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Import-Module (Join-Path $PSScriptRoot 'WindowsSdk.psm1') -Force

if (-not $IdentityName) { $IdentityName = 'Placeholder.WOPR' }
if (-not $Publisher) { $Publisher = 'CN=Placeholder' }
if (-not $PublisherDisplayName) { $PublisherDisplayName = 'Placeholder' }
if ($Version -notmatch '^(\d+\.\d+\.\d+)(-[0-9A-Za-z.-]+)?$') {
    throw "Version '$Version' is not X.Y.Z or X.Y.Z-suffix"
}
$packageVersion = "$($Matches[1]).0"
if ($IdentityName -notmatch '^[A-Za-z0-9.-]{3,50}$') {
    throw "MSIX identity name '$IdentityName' must be 3 to 50 letters, digits, dots and dashes"
}
if ($Publisher -notmatch '^CN=') {
    throw "MSIX publisher '$Publisher' must be a distinguished name such as Partner Center's CN=<GUID>"
}

$BinDir = (Resolve-Path -LiteralPath $BinDir).Path
$null = New-Item -ItemType Directory -Force -Path $OutputDir, $WorkDir
$OutputDir = (Resolve-Path -LiteralPath $OutputDir).Path
$assets = Join-Path $PSScriptRoot '..\icons\msix'
$makeappx = Find-SdkTool 'makeappx.exe'
$makepri = Find-SdkTool 'makepri.exe'

function Invoke-Tool([string] $Tool, [string[]] $Arguments) {
    & $Tool @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "$(Split-Path -Leaf $Tool) $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

# The manifest template with this build's values, XML-escaped, and without its comments.
function Write-Manifest([string] $Path, [string] $Arch) {
    $text = Get-Content -Raw -LiteralPath (Join-Path $PSScriptRoot 'AppxManifest.xml')
    $values = @{
        '@IDENTITY_NAME@' = $IdentityName
        '@PUBLISHER@' = $Publisher
        '@PUBLISHER_DISPLAY_NAME@' = $PublisherDisplayName
        '@VERSION@' = $packageVersion
        '@ARCH@' = $Arch
    }
    foreach ($token in $values.Keys) {
        $text = $text.Replace($token, [Security.SecurityElement]::Escape($values[$token]))
    }
    $doc = [xml]::new()
    $doc.PreserveWhitespace = $true
    $doc.LoadXml($text)
    foreach ($comment in @($doc.SelectNodes('//comment()'))) {
        $null = $comment.ParentNode.RemoveChild($comment)
    }
    if ($doc.OuterXml -match '@[A-Z_]+@') {
        throw "AppxManifest.xml has a token that build-msix.ps1 does not fill: $($Matches[0])"
    }
    $writer = [Xml.XmlWriter]::Create($Path, [Xml.XmlWriterSettings]@{ Encoding = [Text.UTF8Encoding]::new($false) })
    try { $doc.Save($writer) } finally { $writer.Dispose() }
}

# makepri's default configuration, but with one resources.pri that holds every scale and size of
# the icons, rather than separate resource packages.
$priConfig = Join-Path $WorkDir 'priconfig.xml'
Invoke-Tool $makepri @('createconfig', '/cf', $priConfig, '/dq', 'en-US', '/pv', '10.0.0', '/o')
$config = [xml](Get-Content -Raw -LiteralPath $priConfig)
foreach ($node in @($config.SelectNodes('/resources/packaging'))) {
    $null = $node.ParentNode.RemoveChild($node)
}
$config.Save($priConfig)

$bundleDir = Join-Path $WorkDir 'bundle'
$null = New-Item -ItemType Directory -Force -Path $bundleDir
$architectures = [ordered]@{ amd64 = 'x64'; arm64 = 'arm64' } # Go's name, MSIX's name
foreach ($goarch in $architectures.Keys) {
    $arch = $architectures[$goarch]
    $dir = Join-Path $WorkDir $arch
    $null = New-Item -ItemType Directory -Force -Path (Join-Path $dir 'Assets')
    Copy-Item -LiteralPath (Join-Path $BinDir "wopr_${Version}_windows_$goarch.exe") -Destination (Join-Path $dir 'wopr.exe')
    Copy-Item -LiteralPath (Join-Path $BinDir 'LICENSE') -Destination (Join-Path $dir 'LICENSE.txt')
    foreach ($name in 'NOTICE.md', 'THIRD_PARTY_NOTICES.txt', 'README.md') {
        Copy-Item -LiteralPath (Join-Path $BinDir $name) -Destination $dir
    }
    Copy-Item -Path (Join-Path $assets '*.png') -Destination (Join-Path $dir 'Assets')
    $manifest = Join-Path $dir 'AppxManifest.xml'
    Write-Manifest $manifest $arch

    Invoke-Tool $makepri @('new', '/pr', $dir, '/cf', $priConfig, '/mn', $manifest, '/of', (Join-Path $dir 'resources.pri'), '/o')
    $msix = Join-Path $OutputDir "wopr_${Version}_windows_$goarch.msix"
    Invoke-Tool $makeappx @('pack', '/d', $dir, '/p', $msix, '/o')
    Copy-Item -LiteralPath $msix -Destination $bundleDir
}
$bundle = Join-Path $OutputDir "wopr_${Version}_windows.msixbundle"
# An explicit bundle version: without one, makeappx makes one up from the clock.
Invoke-Tool $makeappx @('bundle', '/d', $bundleDir, '/p', $bundle, '/bv', $packageVersion, '/o')

# Unpack the bundle and each package in it again, and check that each holds its architecture's
# program, byte for byte, with the manifest and the resource index.
$check = Join-Path $WorkDir 'check'
Invoke-Tool $makeappx @('unbundle', '/p', $bundle, '/d', $check, '/o')
$inner = @(Get-ChildItem -LiteralPath $check -Recurse -Filter '*.msix')
if ($inner.Count -ne $architectures.Count) {
    throw "the bundle holds $($inner.Count) packages, not $($architectures.Count): $($inner.Name -join ', ')"
}
foreach ($goarch in $architectures.Keys) {
    $package = $inner | Where-Object Name -EQ "wopr_${Version}_windows_$goarch.msix"
    if (-not $package) { throw "the bundle has no package for $goarch" }
    $out = Join-Path $check "unpacked-$goarch"
    Invoke-Tool $makeappx @('unpack', '/p', $package.FullName, '/d', $out, '/o')
    foreach ($name in 'AppxManifest.xml', 'resources.pri', 'wopr.exe', 'LICENSE.txt', 'Assets\Square44x44Logo.scale-100.png') {
        if (-not (Test-Path -LiteralPath (Join-Path $out $name) -PathType Leaf)) { throw "the $goarch package has no $name" }
    }
    $want = (Get-FileHash -LiteralPath (Join-Path $BinDir "wopr_${Version}_windows_$goarch.exe") -Algorithm SHA256).Hash
    $got = (Get-FileHash -LiteralPath (Join-Path $out 'wopr.exe') -Algorithm SHA256).Hash
    if ($got -ne $want) { throw "the $goarch package's wopr.exe is not wopr_${Version}_windows_$goarch.exe" }
    $identity = ([xml](Get-Content -Raw -LiteralPath (Join-Path $out 'AppxManifest.xml'))).Package.Identity
    $gotArch, $gotVersion = $identity.GetAttribute('ProcessorArchitecture'), $identity.GetAttribute('Version')
    if ($gotArch -ne $architectures[$goarch] -or $gotVersion -ne $packageVersion) {
        throw "the $goarch package's identity is $gotArch $gotVersion"
    }
}

foreach ($file in Get-ChildItem -LiteralPath $OutputDir -File) {
    Write-Host "$($file.Name): $($file.Length) bytes, SHA-256 $((Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash)"
}
Write-Host "Identity: $IdentityName, $Publisher ($PublisherDisplayName), version $packageVersion"

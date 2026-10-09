# Installs the MSIX package for this computer's CPU from build-msix.ps1's output, runs
# "wopr --version" through its execution alias, and removes it again. Windows installs only signed
# packages, so this signs a copy with a throwaway certificate whose subject is the manifest's
# publisher, and trusts that certificate on this machine (LocalMachine\TrustedPeople) until the end.
# It needs administrator rights, as on a CI runner. The packages that the Store receives stay
# unsigned.
#
#   pwsh -File packaging/windows/test-msix.ps1 -PackageDir build/msix -Version 1.2.3
[CmdletBinding()]
param(
    [Parameter(Mandatory)] [string] $PackageDir,
    [Parameter(Mandatory)] [string] $Version
)

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest
Import-Module (Join-Path $PSScriptRoot 'WindowsSdk.psm1') -Force

$goarch = switch ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture) {
    'X64' { 'amd64' }
    'Arm64' { 'arm64' }
    default { throw "no Windows build for $_" }
}
$package = (Resolve-Path -LiteralPath (Join-Path $PackageDir "wopr_${Version}_windows_$goarch.msix")).Path
$alias = Join-Path $env:LOCALAPPDATA 'Microsoft\WindowsApps\wopr.exe'
$work = Join-Path ([IO.Path]::GetTempPath()) "wopr-msix-test-$([guid]::NewGuid())"
$null = New-Item -ItemType Directory -Path $work

# The Appx cmdlets run in Windows PowerShell, where they always load; PowerShell 7 can load them
# only through a compatibility session on some versions of Windows. The command reads its
# arguments from WOPR_* environment variables, so nothing needs quoting.
function Invoke-WindowsPowerShell([string] $Command) {
    $out = & powershell.exe -NoProfile -NonInteractive -Command "`$ErrorActionPreference = 'Stop'; `$ProgressPreference = 'SilentlyContinue'; $Command"
    if ($LASTEXITCODE -ne 0) { throw "Windows PowerShell failed ($LASTEXITCODE): $Command" }
    return $out
}

function Assert([bool] $Condition, [string] $Message) {
    if (-not $Condition) { throw "FAILED: $Message" }
    Write-Host "ok: $Message"
}

# The identity to install and to sign for, from the package's own manifest.
Add-Type -AssemblyName System.IO.Compression.FileSystem
$zip = [IO.Compression.ZipFile]::OpenRead($package)
try {
    $reader = [IO.StreamReader]::new($zip.GetEntry('AppxManifest.xml').Open())
    try { $identity = ([xml]$reader.ReadToEnd()).Package.Identity } finally { $reader.Dispose() }
} finally {
    $zip.Dispose()
}
$name = $identity.GetAttribute('Name')
$publisher = $identity.GetAttribute('Publisher')
Write-Host "Package: $package ($name, $publisher, $($identity.GetAttribute('Version')), $($identity.GetAttribute('ProcessorArchitecture')))"

# A self-signed code-signing certificate for the publisher, valid for a day.
$rsa = [Security.Cryptography.RSA]::Create(2048)
$request = [Security.Cryptography.X509Certificates.CertificateRequest]::new(
    [Security.Cryptography.X509Certificates.X500DistinguishedName]::new($publisher), $rsa,
    [Security.Cryptography.HashAlgorithmName]::SHA256, [Security.Cryptography.RSASignaturePadding]::Pkcs1)
$request.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509BasicConstraintsExtension]::new($false, $false, 0, $true))
$request.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509KeyUsageExtension]::new(
        [Security.Cryptography.X509Certificates.X509KeyUsageFlags]::DigitalSignature, $true))
$oids = [Security.Cryptography.OidCollection]::new()
$null = $oids.Add([Security.Cryptography.Oid]::new('1.3.6.1.5.5.7.3.3')) # code signing
$request.CertificateExtensions.Add([Security.Cryptography.X509Certificates.X509EnhancedKeyUsageExtension]::new($oids, $false))
$cert = $request.CreateSelfSigned([DateTimeOffset]::UtcNow.AddMinutes(-5), [DateTimeOffset]::UtcNow.AddDays(1))
$password = [guid]::NewGuid().ToString()
$pfx = Join-Path $work 'test.pfx'
[IO.File]::WriteAllBytes($pfx, $cert.Export([Security.Cryptography.X509Certificates.X509ContentType]::Pfx, $password))
$public = [Security.Cryptography.X509Certificates.X509Certificate2]::new($cert.Export([Security.Cryptography.X509Certificates.X509ContentType]::Cert))

$signed = Join-Path $work (Split-Path -Leaf $package)
Copy-Item -LiteralPath $package -Destination $signed
$signtool = Find-SdkTool 'signtool.exe'
& $signtool sign /fd SHA256 /f $pfx /p $password $signed
Assert ($LASTEXITCODE -eq 0) "signtool signed a copy for $publisher"

$store = [Security.Cryptography.X509Certificates.X509Store]::new('TrustedPeople', 'LocalMachine')
$store.Open('ReadWrite')
$installed = $false
try {
    $store.Add($public)
    $env:WOPR_MSIX = $signed
    $env:WOPR_MSIX_NAME = $name
    Invoke-WindowsPowerShell 'Add-AppxPackage -Path $env:WOPR_MSIX'
    $installed = $true
    $fullName = Invoke-WindowsPowerShell '(Get-AppxPackage -Name $env:WOPR_MSIX_NAME).PackageFullName'
    Assert ([bool]$fullName) "Windows installed $fullName"

    Assert (Test-Path -LiteralPath $alias) "the execution alias is $alias"
    $out = & $alias --version
    Assert ($LASTEXITCODE -eq 0 -and "$out" -like "wopr v$Version (*") "wopr --version through the alias prints v${Version}: $out"

    Invoke-WindowsPowerShell 'Get-AppxPackage -Name $env:WOPR_MSIX_NAME | Remove-AppxPackage'
    $installed = $false
    for ($i = 0; $i -lt 30 -and (Test-Path -LiteralPath $alias); $i++) { Start-Sleep -Seconds 1 }
    Assert (-not (Test-Path -LiteralPath $alias)) 'removing the package removed the alias'
    $left = Invoke-WindowsPowerShell '@(Get-AppxPackage -Name $env:WOPR_MSIX_NAME).Count'
    Assert ("$left" -eq '0') 'Windows no longer lists the package'
} finally {
    if ($installed) {
        Invoke-WindowsPowerShell 'Get-AppxPackage -Name $env:WOPR_MSIX_NAME | Remove-AppxPackage'
    }
    $store.Remove($public)
    $store.Dispose()
    Remove-Item -LiteralPath $work -Recurse -Force
}
Write-Host 'The MSIX package installs, runs through its alias, and uninstalls cleanly.'

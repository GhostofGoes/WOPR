# Finds a tool of the Windows SDK (makeappx.exe, makepri.exe, signtool.exe) for build-msix.ps1 and
# test-msix.ps1: the newest SDK's build for this computer's CPU, or else its x64 build, which Arm64
# Windows 11 runs too. The SDK's folder comes from the registry, as the SDK records it.

function Find-SdkTool {
    [CmdletBinding()]
    [OutputType([string])]
    param([Parameter(Mandatory)] [string] $Name)

    $roots = [Collections.Generic.List[string]]::new()
    foreach ($key in 'HKLM:\SOFTWARE\Microsoft\Windows Kits\Installed Roots',
        'HKLM:\SOFTWARE\WOW6432Node\Microsoft\Windows Kits\Installed Roots') {
        $item = Get-ItemProperty -LiteralPath $key -ErrorAction SilentlyContinue
        if ($null -ne $item -and $item.PSObject.Properties['KitsRoot10']) {
            $roots.Add($item.KitsRoot10)
        }
    }
    $roots.Add((Join-Path ${env:ProgramFiles(x86)} 'Windows Kits\10'))

    $arches = if ([Runtime.InteropServices.RuntimeInformation]::OSArchitecture -eq 'Arm64') { 'arm64', 'x64' } else { 'x64' }
    $found = foreach ($root in $roots | Select-Object -Unique) {
        $bin = Join-Path $root 'bin'
        if (-not (Test-Path -LiteralPath $bin)) { continue }
        foreach ($dir in Get-ChildItem -LiteralPath $bin -Directory | Where-Object Name -Match '^\d+\.\d+\.\d+\.\d+$') {
            $rank = 0
            foreach ($arch in $arches) {
                $tool = Join-Path $dir.FullName "$arch\$Name"
                if (Test-Path -LiteralPath $tool -PathType Leaf) {
                    [pscustomobject]@{ Version = [version]$dir.Name; Rank = $rank; Path = $tool }
                }
                $rank++
            }
        }
    }
    $best = $found | Sort-Object -Property @{ Expression = 'Version'; Descending = $true }, Rank | Select-Object -First 1
    if ($null -eq $best) {
        throw "$Name is in no Windows SDK under $($roots -join ', '): install the Windows SDK"
    }
    Write-Host "$Name`: $($best.Path)"
    return $best.Path
}

Export-ModuleMember -Function Find-SdkTool

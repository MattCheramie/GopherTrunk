param(
    [string]$DeviceInstanceId = "USB\VID_1D50&PID_60A1\AIRSPY_SN:35AC63DC2D645A4F",
    [string]$AirspyInfoPath = "tools\libairspy-v1.0.10\airspy_host_tools_win32_x86_x64_v1_0_10\x86\airspy_info.exe",
    [Alias("RenumerationTimeoutSeconds")]
    [int]$ReenumerationTimeoutSeconds = 30
)

$ErrorActionPreference = "Stop"
$ScriptVersion = "2026-05-26b"

function Test-IsAdmin {
    $identity = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Invoke-AirspyInfo {
    param([string]$Path)

    if (-not (Test-Path -LiteralPath $Path)) {
        throw "airspy_info.exe not found at: $Path"
    }

    Write-Host "=== Running airspy_info: $Path ==="
    & $Path
    Write-Host "airspy_info exit=$LASTEXITCODE"
}

function Resolve-RepoPath {
    param([string]$RelativePath)

    $repoRoot = Split-Path -Parent $PSScriptRoot
    return Join-Path $repoRoot $RelativePath
}

function Resolve-AirspyInstanceId {
    param([string]$PreferredInstanceId)

    try {
        Get-PnpDevice -InstanceId $PreferredInstanceId -ErrorAction Stop | Out-Null
        return $PreferredInstanceId
    }
    catch {
        # Continue to wildcard discovery.
    }

    $normalized = $PreferredInstanceId -replace "\\\\", "\\"
    if ($normalized -ne $PreferredInstanceId) {
        try {
            Get-PnpDevice -InstanceId $normalized -ErrorAction Stop | Out-Null
            return $normalized
        }
        catch {
            # Continue to wildcard discovery.
        }
    }

    $matches = Get-PnpDevice | Where-Object {
        $_.InstanceId -like "USB\VID_1D50&PID_60A1\*"
    }

    if (-not $matches) {
        throw "No connected Airspy device matched VID_1D50&PID_60A1."
    }

    if ($matches.Count -gt 1) {
        throw "Multiple Airspy devices found. Re-run with -DeviceInstanceId using one of:`n$($matches.InstanceId -join "`n")"
    }

    return $matches[0].InstanceId
}

function Wait-ForDevicePresent {
    param(
        [string]$InstanceId,
        [int]$TimeoutSeconds
    )

    $deadline = (Get-Date).AddSeconds($TimeoutSeconds)
    while ((Get-Date) -lt $deadline) {
        $dev = Get-PnpDevice -InstanceId $InstanceId -ErrorAction SilentlyContinue
        if ($null -ne $dev -and $dev.Present) {
            return $dev.InstanceId
        }

        $matches = Get-PnpDevice | Where-Object {
            $_.InstanceId -like "USB\VID_1D50&PID_60A1\*" -and $_.Present
        }
        if ($matches.Count -eq 1) {
            return $matches[0].InstanceId
        }
        if ($matches.Count -gt 1) {
            throw "Multiple present Airspy devices found during re-enumeration. Re-run with -DeviceInstanceId using one of:`n$($matches.InstanceId -join "`n")"
        }

        Start-Sleep -Milliseconds 500
    }

    return $null
}

if (-not (Test-IsAdmin)) {
    throw "Run this script from an elevated PowerShell session (Run as Administrator)."
}

$resolvedInstanceId = Resolve-AirspyInstanceId -PreferredInstanceId $DeviceInstanceId
$resolvedAirspyInfoPath = Resolve-RepoPath -RelativePath $AirspyInfoPath

Write-Host "ScriptVersion: $ScriptVersion"
Write-Host "Effective ReenumerationTimeoutSeconds: $ReenumerationTimeoutSeconds"
Write-Host "Using DeviceInstanceId: $resolvedInstanceId"
Write-Host "Using airspy_info path: $resolvedAirspyInfoPath"

Write-Host "=== Device status (before) ==="
 $before = Get-PnpDevice -InstanceId $resolvedInstanceId | Select-Object Status, Class, FriendlyName, Service, Present, InstanceId
 $before | Format-List

$isPresent = $false
if ($null -ne $before.Present) {
    try {
        $isPresent = [System.Convert]::ToBoolean($before.Present)
    }
    catch {
        $isPresent = ($before.Present.ToString().Trim().ToLowerInvariant() -eq "true")
    }
}
Write-Host "Evaluated Present: $isPresent"

if ($isPresent) {
    Write-Host "=== Attempting restart-device ==="
    pnputil /restart-device "$resolvedInstanceId"
}
else {
    Write-Host "=== Device is not currently present; skipping restart and waiting for re-enumeration/replug ==="
}

Write-Host "=== Scanning for devices ==="
pnputil /scan-devices

Write-Host "=== Waiting for re-enumeration (up to $ReenumerationTimeoutSeconds s) ==="
$presentInstanceId = Wait-ForDevicePresent -InstanceId $resolvedInstanceId -TimeoutSeconds $ReenumerationTimeoutSeconds
if (-not $presentInstanceId) {
    throw "Device did not re-enumerate within timeout. Unplug/replug Airspy, then rerun this script with -ReenumerationTimeoutSeconds 60."
}

$resolvedInstanceId = $presentInstanceId
Write-Host "Re-enumerated DeviceInstanceId: $resolvedInstanceId"

Write-Host "=== Device status (after restart) ==="
Get-PnpDevice -InstanceId $resolvedInstanceId | Select-Object Status, Class, FriendlyName, Service, Present | Format-List

Invoke-AirspyInfo -Path $resolvedAirspyInfoPath

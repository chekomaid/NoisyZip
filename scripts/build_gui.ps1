param(
    [string[]]$WailsArgs = @("-clean")
)

$ErrorActionPreference = "Stop"

function Has-WailsOutputFlag {
    param([string[]]$Args)
    foreach ($arg in $Args) {
        if ($arg -eq "-o") { return $true }
        if ($arg -like "-o=*") { return $true }
    }
    return $false
}

function Has-WailsCleanFlag {
    param([string[]]$Args)
    foreach ($arg in $Args) {
        if ($arg -eq "-clean") { return $true }
    }
    return $false
}

function Has-WailsPlatformFlag {
    param([string[]]$Args)
    foreach ($arg in $Args) {
        if ($arg -eq "-platform") { return $true }
        if ($arg -like "-platform=*") { return $true }
    }
    return $false
}

function Get-WailsPlatformValue {
    param([string[]]$Args)
    for ($i = 0; $i -lt $Args.Length; $i++) {
        $arg = $Args[$i]
        if ($arg -eq "-platform" -and ($i + 1) -lt $Args.Length) {
            return $Args[$i + 1]
        }
        if ($arg -like "-platform=*") {
            return $arg.Substring(10)
        }
    }
    return $null
}

function Ensure-GuiTags {
    param([string[]]$Args)
    $out = @()
    $handled = $false
    for ($i = 0; $i -lt $Args.Length; $i++) {
        $arg = $Args[$i]
        if ($arg -eq "-tags" -and ($i + 1) -lt $Args.Length) {
            $tags = $Args[$i + 1]
            if ($tags -notmatch "(^|[ ,])gui([ ,]|$)") {
                $tags = "$tags,gui"
            }
            $out += @("-tags", $tags)
            $i++
            $handled = $true
            continue
        }
        if ($arg -like "-tags=*") {
            $tags = $arg.Substring(6)
            if ($tags -notmatch "(^|[ ,])gui([ ,]|$)") {
                $tags = "$tags,gui"
            }
            $out += "-tags=$tags"
            $handled = $true
            continue
        }
        $out += $arg
    }
    if (-not $handled) {
        $out += @("-tags", "gui")
    }
    return $out
}

function Ensure-CleanFlag {
    param([string[]]$Args)
    if (Has-WailsCleanFlag -Args $Args) {
        return $Args
    }
    $out = @("-clean")
    if ($null -ne $Args) {
        $out += $Args
    }
    return $out
}

function Remove-WailsFlagWithValue {
    param([string[]]$Args, [string]$Flag)
    $out = @()
    for ($i = 0; $i -lt $Args.Length; $i++) {
        $arg = $Args[$i]
        if ($arg -eq $Flag) {
            $i++
            continue
        }
        if ($arg -like "$Flag=*") {
            continue
        }
        $out += $arg
    }
    return $out
}

$WailsArgs = Ensure-GuiTags -Args $WailsArgs
$WailsArgs = Ensure-CleanFlag -Args $WailsArgs

$prevGOFLAGS = $env:GOFLAGS
try {
    if (Has-WailsPlatformFlag -Args $WailsArgs) {
        if (-not (Has-WailsOutputFlag -Args $WailsArgs)) {
            $platform = Get-WailsPlatformValue -Args $WailsArgs
            if ($null -ne $platform -and $platform.Trim() -ne "") {
                $parts = $platform.Split("/")
                if ($parts.Length -eq 2) {
                    $os = $parts[0].ToLower()
                    $arch = $parts[1].ToLower()
                    $out = "noisyzip-gui-$os-$arch"
                    if ($os -eq "windows") {
                        $out += ".exe"
                    }
                    $WailsArgs += @("-o", $out)
                }
            }
        }
        Write-Host ("Building GUI with: wails build {0}" -f ($WailsArgs -join " "))
        wails build @WailsArgs
        return
    }

    $baseArgs = @("-tags", "gui")
    $cleanArgs = @("-clean")

    $targets = @(
        @{ platform = "windows/amd64"; output = "noisyzip-gui-windows-amd64.exe" },
        @{ platform = "windows/arm64"; output = "noisyzip-gui-windows-arm64.exe" }
    )

    for ($i = 0; $i -lt $targets.Count; $i++) {
        $t = $targets[$i]
        $buildArgs = @()
        if ($i -eq 0) {
            $buildArgs += $cleanArgs
        }
        $buildArgs += $baseArgs
        $buildArgs += @("-platform", $t.platform, "-o", $t.output)
        Write-Host ("Building GUI with: wails build {0}" -f ($buildArgs -join " "))
        wails build @buildArgs
    }
}
finally {
    if ($null -ne $prevGOFLAGS) {
        $env:GOFLAGS = $prevGOFLAGS
    } else {
        Remove-Item Env:GOFLAGS -ErrorAction SilentlyContinue
    }
}

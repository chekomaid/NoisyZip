param(
    [string]$OutDir = "build/cli",
    [string]$Name = "noisyzip"
)

$ErrorActionPreference = "Stop"
$ldflags = "-X noisyzip/internal/cli.Version=2.1"

function Resolve-RepoRoot {
    $root = Join-Path $PSScriptRoot ".."
    return (Resolve-Path $root).Path
}

$repoRoot = Resolve-RepoRoot
$outPath = Join-Path $repoRoot $OutDir
New-Item -ItemType Directory -Path $outPath -Force | Out-Null

$targets = @(
    @{ os = "windows"; arch = "amd64"; ext = ".exe" },
    @{ os = "windows"; arch = "arm64"; ext = ".exe" },
    @{ os = "linux"; arch = "amd64"; ext = "" },
    @{ os = "linux"; arch = "arm64"; ext = "" },
    @{ os = "darwin"; arch = "amd64"; ext = "" },
    @{ os = "darwin"; arch = "arm64"; ext = "" }
)

$prevGOOS = $env:GOOS
$prevGOARCH = $env:GOARCH
$prevCGO = $env:CGO_ENABLED

Push-Location $repoRoot
try {
    foreach ($t in $targets) {
        $env:GOOS = $t.os
        $env:GOARCH = $t.arch
        $env:CGO_ENABLED = "0"

        $outfile = Join-Path $outPath ("{0}-{1}-{2}{3}" -f $Name, $t.os, $t.arch, $t.ext)
        Write-Host ("Building {0}/{1} -> {2}" -f $t.os, $t.arch, $outfile)
        go build -ldflags $ldflags -o $outfile .
    }
}
finally {
    if ($null -ne $prevGOOS) { $env:GOOS = $prevGOOS } else { Remove-Item Env:GOOS -ErrorAction SilentlyContinue }
    if ($null -ne $prevGOARCH) { $env:GOARCH = $prevGOARCH } else { Remove-Item Env:GOARCH -ErrorAction SilentlyContinue }
    if ($null -ne $prevCGO) { $env:CGO_ENABLED = $prevCGO } else { Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue }
    Pop-Location
}

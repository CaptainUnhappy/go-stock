$ErrorActionPreference = "Stop"

function Invoke-NativeCommand {
    param(
        [Parameter(Mandatory = $true)]
        [scriptblock]$Command,
        [Parameter(Mandatory = $true)]
        [string]$Name
    )
    & $Command
    if ($LASTEXITCODE -ne 0) {
        throw "$Name failed with exit code $LASTEXITCODE"
    }
}

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$wailsConfig = Get-Content (Join-Path $repoRoot "wails.json") -Raw | ConvertFrom-Json
$version = $wailsConfig.info.productVersion
if ([string]::IsNullOrWhiteSpace($version)) {
    $version = "dev"
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$releaseDir = Join-Path $repoRoot ("build\releases\v{0}-{1}" -f $version, $stamp)

Push-Location $repoRoot
$oldGoCache = $env:GOCACHE
$oldGoModCache = $env:GOMODCACHE
$oldGoPath = $env:GOPATH
$oldGoTelemetry = $env:GOTELEMETRY
$oldGoTelemetryDir = $env:GOTELEMETRYDIR
try {
    $repoGoCache = Join-Path $repoRoot ".gocache"
    $repoGoModCache = Join-Path $repoRoot ".gomodcache"
    $repoGoPath = Join-Path $repoRoot ".gopath"
    $repoGoTelemetryDir = Join-Path $repoRoot ".gotelemetry"
    New-Item -ItemType Directory -Force -Path $repoGoCache | Out-Null
    New-Item -ItemType Directory -Force -Path $repoGoModCache | Out-Null
    New-Item -ItemType Directory -Force -Path $repoGoPath | Out-Null
    New-Item -ItemType Directory -Force -Path $repoGoTelemetryDir | Out-Null
    $env:GOCACHE = $repoGoCache
    $env:GOMODCACHE = $repoGoModCache
    $env:GOPATH = $repoGoPath
    $env:GOTELEMETRY = "off"
    $env:GOTELEMETRYDIR = $repoGoTelemetryDir

    Invoke-NativeCommand -Name "go-stock-cli build" -Command { go build -o (Join-Path $repoRoot "go-stock-cli.exe") .\cmd\go-stock-cli }
    Invoke-NativeCommand -Name "wails build" -Command { wails build --clean --platform windows/amd64 }
    New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null
    Copy-Item (Join-Path $repoRoot "build\bin\go-stock.exe") (Join-Path $releaseDir "go-stock.exe") -Force
    Copy-Item (Join-Path $repoRoot "go-stock-cli.exe") (Join-Path $releaseDir "go-stock-cli.exe") -Force
    Write-Host "Built release: $releaseDir\go-stock.exe"
    Write-Host "Built release: $releaseDir\go-stock-cli.exe"
    Write-Host "Runtime data is stored outside build output under the go-stock user data directory."
}
finally {
    $env:GOCACHE = $oldGoCache
    $env:GOMODCACHE = $oldGoModCache
    $env:GOPATH = $oldGoPath
    $env:GOTELEMETRY = $oldGoTelemetry
    $env:GOTELEMETRYDIR = $oldGoTelemetryDir
    Pop-Location
}

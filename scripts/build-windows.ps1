$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$wailsConfig = Get-Content (Join-Path $repoRoot "wails.json") -Raw | ConvertFrom-Json
$version = $wailsConfig.info.productVersion
if ([string]::IsNullOrWhiteSpace($version)) {
    $version = "dev"
}

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$releaseDir = Join-Path $repoRoot ("build\releases\v{0}-{1}" -f $version, $stamp)

Push-Location $repoRoot
try {
    wails build --clean --platform windows/amd64
    New-Item -ItemType Directory -Force -Path $releaseDir | Out-Null
    Copy-Item (Join-Path $repoRoot "build\bin\go-stock.exe") (Join-Path $releaseDir "go-stock.exe") -Force
    Write-Host "Built release: $releaseDir\go-stock.exe"
    Write-Host "Runtime data is stored outside build output under the go-stock user data directory."
}
finally {
    Pop-Location
}

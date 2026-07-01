param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]] $CliArgs
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$cacheDir = Join-Path $repoRoot ".gocache"
$goAppDataDir = Join-Path $repoRoot ".go-appdata"

if (-not (Test-Path -LiteralPath $cacheDir)) {
  New-Item -ItemType Directory -Path $cacheDir | Out-Null
}
if (-not (Test-Path -LiteralPath $goAppDataDir)) {
  New-Item -ItemType Directory -Path $goAppDataDir | Out-Null
}

if ([string]::IsNullOrWhiteSpace($env:GO_STOCK_HOME) -and -not [string]::IsNullOrWhiteSpace($env:APPDATA)) {
  $env:GO_STOCK_HOME = Join-Path $env:APPDATA "go-stock"
}

$env:GOCACHE = $cacheDir
$env:GOTELEMETRY = "off"
$env:APPDATA = $goAppDataDir
Push-Location $repoRoot
try {
  & go run ./cmd/go-stock-cli @CliArgs
  exit $LASTEXITCODE
} finally {
  Pop-Location
}

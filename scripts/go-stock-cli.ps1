param(
  [Parameter(ValueFromRemainingArguments = $true)]
  [string[]] $CliArgs
)

$ErrorActionPreference = "Stop"

$repoRoot = Resolve-Path (Join-Path $PSScriptRoot "..")
$cacheDir = Join-Path $repoRoot ".gocache"

if (-not (Test-Path -LiteralPath $cacheDir)) {
  New-Item -ItemType Directory -Path $cacheDir | Out-Null
}

$env:GOCACHE = $cacheDir
Push-Location $repoRoot
try {
  & go run ./cmd/go-stock-cli @CliArgs
  exit $LASTEXITCODE
} finally {
  Pop-Location
}

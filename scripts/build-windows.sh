#! /bin/bash

echo -e "Start running the script..."
cd "$(dirname "$0")/.."

echo -e "Start building the app for windows platform..."
wails build --clean --platform windows/amd64

VERSION=$(grep -o '"productVersion"[[:space:]]*:[[:space:]]*"[^"]*"' wails.json | sed -E 's/.*"([^"]+)"/\1/')
if [ -z "$VERSION" ]; then
  VERSION="dev"
fi
STAMP=$(date +"%Y%m%d-%H%M%S")
RELEASE_DIR="build/releases/v${VERSION}-${STAMP}"
mkdir -p "$RELEASE_DIR"
cp build/bin/go-stock.exe "$RELEASE_DIR/go-stock.exe"
echo -e "Built release: ${RELEASE_DIR}/go-stock.exe"

echo -e "End running the script!"

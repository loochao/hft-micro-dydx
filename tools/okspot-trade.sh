#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-okspot-trade.$dt" ./recorders/okex-usdtspot-trade

git add -A
git commit -m "build hft-recorder-okspot-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-okspot-trade.$dt"

echo "tokyo3"
rsync -avx --progress "./dist/hft-recorder-okspot-trade.$dt" luchao:/usr/local/bin/

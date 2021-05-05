#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-kcperp-trade.$dt" ./recorders/kcperp-trade

git add -A
git commit -m "build hft-recorder-kcperp-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-kcperp-trade.$dt"

echo "tokyo3"
rsync -avx --progress "./dist/hft-recorder-kcperp-trade.$dt" lochao:/usr/local/bin/

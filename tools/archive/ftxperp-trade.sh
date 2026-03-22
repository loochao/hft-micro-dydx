#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-ftxperp-trade.$dt" ./recorders/ftx-usdfuture-trade

git add -A
git commit -m "build hft-recorder-ftxperp-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-ftxperp-trade.$dt"

echo "ir"
rsync -avx --progress "./dist/hft-recorder-ftxperp-trade.$dt" ir:/opt/bin/

#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-bnswap-trade.$dt" ./recorders/bnswap-trade

git add -A
git commit -m "build hft-recorder-bnswap-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-bnswap-trade.$dt"

echo "tokyo2"
rsync -avx --progress "./dist/hft-recorder-bnswap-trade.$dt" tokyo2:/usr/local/bin/

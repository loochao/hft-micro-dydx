#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-bnspot-trade.$dt" ./recorders/bnspot-trade

git add -A
git commit -m "build hft-recorder-bnspot-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-bnspot-trade.$dt"

echo "tokyo3"
rsync -avx --progress "./dist/hft-recorder-bnspot-trade.$dt" tokyo3:/usr/local/bin/

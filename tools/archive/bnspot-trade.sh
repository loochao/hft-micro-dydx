#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-bnspot-trade.$dt" ./recorders/bnspot-trade

git add -A
git commit -m "build hft-recorder-bnspot-trade.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-bnspot-trade.$dt"

echo "data01"
rsync -avx --progress "./dist/hft-recorder-bnspot-trade.$dt" ir:/mnt/d1/data-tmp/bin/
ssh ir "rsync -avx --progress /mnt/d1/data-tmp/bin/hft-recorder-bnspot-trade.$dt data01:/usr/local/bin/"


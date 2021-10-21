#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnspot-usd.$dt" ./recorders/bnspot-usd

git add -A
git commit -m "build bnspot-usd.$dt"
git push origin master

chmod 755 "./dist/bnspot-usd.$dt"

echo "hk11"
rsync -avx --progress "./dist/bnspot-usd.$dt" hk11:/usr/local/bin/


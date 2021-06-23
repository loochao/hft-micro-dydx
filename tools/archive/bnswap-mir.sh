#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-bnswap-mir.$dt" ./applications/bnswap-mir

git add -A
git commit -m "build hft-bnswap-mir.$dt"
git push origin master

chmod 755 "./dist/hft-bnswap-mir.$dt"

echo "tokyo3"
rsync -avx --progress "./dist/hft-bnswap-mir.$dt" luchao:/usr/local/bin/

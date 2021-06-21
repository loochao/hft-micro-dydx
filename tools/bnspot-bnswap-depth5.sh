#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnspot-bnswap-depth5.$dt" ./recorders/bnspot-bnswap-depth5

git add -A
git commit -m "build bnspot-bnswap-depth5.$dt"
git push origin master

chmod 755 "./dist/bnspot-bnswap-depth5.$dt"

echo "hk04"
rsync -avx --progress "./dist/bnspot-bnswap-depth5.$dt" hk04:/usr/local/bin/


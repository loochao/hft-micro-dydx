#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnswap-depth5-leadlag.$dt" ./recorders/bnswap-depth5-leadlag

git add -A
git commit -m "build bnswap-depth5-leadlag.$dt"
git push origin master

chmod 755 "./dist/bnswap-depth5-leadlag.$dt"

echo "hk01"
rsync -avx --progress "./dist/bnswap-depth5-leadlag.$dt" hk02:/usr/local/bin/


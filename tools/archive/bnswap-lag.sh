#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswap-lag/init.go

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-bnswap-lag.$dt" ./applications/bnswap-lag

git add -A
git commit -m "build hft-mirco-bnswap-lag.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-bnswap-lag.$dt"

echo "ff05"
rsync -avx --progress "./dist/hft-mirco-bnswap-lag.$dt" ff05:/usr/local/bin/

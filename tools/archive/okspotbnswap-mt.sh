#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/okspotbnswap-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-okspotbnswap-mt.$dt" ./applications/okspotbnswap-mt

git add -A
git commit -m "build hft-mirco-okspotbnswap-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-okspotbnswap-mt.$dt"


echo "ff05"
rsync -avx --progress "./dist/hft-mirco-okspotbnswap-mt.$dt" ff05:/usr/local/bin/

#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/kcperpbnswap-usdt-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-kcperpbnswap-opt-mt.$dt" ./applications/kcperpbnswap-usdt-mt

git add -A
git commit -m "build hft-mirco-kcperpbnswap-opt-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-kcperpbnswap-opt-mt.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-kcperpbnswap-opt-mt.$dt" wenzhe:/usr/local/bin/

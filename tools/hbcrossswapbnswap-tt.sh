#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/hbcrossswapbnswap-mt/init.go

env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-hbcrossswapbnswap-tt.$dt" ./applications/hbcrossswapbnswap-mt

git add -A
git commit -m "build hft-mirco-hbcrossswapbnswap-tt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-hbcrossswapbnswap-tt.$dt"

echo "pd02"
rsync -avx --progress "./dist/hft-mirco-hbcrossswapbnswap-tt.$dt" pd02:/usr/local/bin/

echo "tokyo1"
rsync -avx --progress "./dist/hft-mirco-hbcrossswapbnswap-tt.$dt" tokyo1:/usr/local/bin/


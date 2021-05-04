#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnspotswap-routed-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-bnswapspot-routed-mt.$dt" ./applications/bnspotswap-routed-mt

git add -A
git commit -m "build hft-mirco-bnswapspot-routed-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-bnswapspot-routed-mt.$dt"

echo "pd02"
rsync -avx --progress "./dist/hft-mirco-bnswapspot-routed-mt.$dt" pd02:/usr/local/bin/

echo "ff04"
rsync -avx --progress "./dist/hft-mirco-bnswapspot-routed-mt.$dt" ff04:/usr/local/bin/

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-bnswapspot-routed-mt.$dt" wenzhe:/usr/local/bin/

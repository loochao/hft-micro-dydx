#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswap-lob/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-bnswap-lob.$dt" ./applications/bnswap-lob

git add -A
git commit -m "build hft-mirco-bnswap-lob.$dt"
#git push origin master

chmod 755 "./dist/hft-mirco-bnswap-lob.$dt"


echo "ff04"
rsync -avx --progress "./dist/hft-mirco-bnswap-lob.$dt" ff04:/usr/local/bin/


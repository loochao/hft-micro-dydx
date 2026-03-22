#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswap-ll/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-bnswap-ll.$dt" ./applications/bnswap-ll

git add -A
git commit -m "build hft-mirco-bnswap-ll.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-bnswap-ll.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-bnswap-ll.$dt" wenzhe:/usr/local/bin/

#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswapspot-tt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/bnswapspot-tt.$dt" ./applications/bnswapspot-tt

git add -A
git commit -m "build bnswapspot-tt.$dt"
git push origin master

chmod 755 "./dist/bnswapspot-tt.$dt"

echo "pd02"
rsync -avx --progress "./dist/bnswapspot-tt.$dt" pd02:/usr/local/bin/

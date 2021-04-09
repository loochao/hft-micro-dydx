#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswapspot-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/bnswapspot-mt.$dt" ./applications/bnswapspot-mt

git add -A
git commit -m "build bnswapspot-mt.$dt"
git push origin master

chmod 755 "./dist/bnswapspot-mt.$dt"

echo "pd02"
rsync -avx --progress "./dist/bnswapspot-mt.$dt" pd02:/usr/local/bin/

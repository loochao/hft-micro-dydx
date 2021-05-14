#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/opt-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-opt-mt.$dt" ./applications/opt-mt

git add -A
git commit -m "build hft-mirco-opt-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-opt-mt.$dt"

echo "pd02"
rsync -avx --progress "./dist/hft-mirco-opt-mt.$dt" ff04:/usr/local/bin/

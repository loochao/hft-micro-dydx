#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/opt-grid/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-opt-grid.$dt" ./applications/opt-grid

git add -A
git commit -m "build hft-mirco-opt-grid.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-opt-grid.$dt"

echo "ff05"
rsync -avx --progress "./dist/hft-mirco-opt-grid.$dt" ff05:/usr/local/bin/

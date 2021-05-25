#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/opt-tt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-opt-tt.$dt" ./applications/opt-tt

git add -A
git commit -m "build hft-mirco-opt-tt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-opt-tt.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-opt-tt.$dt" wenzhe:/usr/local/bin/

echo "pd02"
rsync -avx --progress "./dist/hft-mirco-opt-tt.$dt" pd02:/usr/local/bin/

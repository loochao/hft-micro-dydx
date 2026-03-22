#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/coin-tt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/coin-tt.arm64.$dt" ./applications/coin-tt

git add -A
git commit -m "build coin-tt.$dt"
git push origin master

chmod 755 "./dist/coin-tt.arm64.$dt"

echo "arm1"
rsync -avx --progress "./dist/coin-tt.arm64.$dt" arm1:/usr/local/bin/

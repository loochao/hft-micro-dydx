#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/bnswap-mm/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-recorder-bnswap-depth20.$dt" ./applications/bnswap-mm

git add -A
git commit -m "build hft-recorder-bnswap-depth20.$dt"
git push origin master

chmod 755 "./dist/hft-recorder-bnswap-depth20.$dt"

echo "tokyo1"
rsync -avx --progress "./dist/hft-recorder-bnswap-depth20.$dt" wenzhe:/usr/local/bin/

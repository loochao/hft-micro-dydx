#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/kcperpspot-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-kcperpspot-mt.$dt" ./applications/kcperpspot-mt

git add -A
git commit -m "build hft-mirco-kcperpspot-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-kcperpspot-mt.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-kcperpspot-mt.$dt" wenzhe:/usr/local/bin/

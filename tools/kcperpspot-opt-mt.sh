#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/kcperpspot-opt-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-kcperpspot-opt-mt.$dt" ./applications/kcperpspot-opt-mt

git add -A
git commit -m "build hft-mirco-kcperpspot-opt-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-kcperpspot-opt-mt.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-kcperpspot-opt-mt.$dt" wenzhe:/usr/local/bin/

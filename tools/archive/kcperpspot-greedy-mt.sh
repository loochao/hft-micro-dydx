#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/kcperpspot-greedy-mt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-kcperpspot-greedy-mt.$dt" ./applications/kcperpspot-greedy-mt

git add -A
git commit -m "build hft-mirco-kcperpspot-greedy-mt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-kcperpspot-greedy-mt.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-kcperpspot-greedy-mt.$dt" wenzhe:/usr/local/bin/

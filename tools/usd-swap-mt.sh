#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-swap-mt/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-swap-mt.arm64.$dt" ./applications/usd-swap-mt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-swap-mt.amd64.$dt" ./applications/usd-swap-mt

chmod 755 "./dist/usd-swap-mt.amd64.$dt"
chmod 755 "./dist/usd-swap-mt.arm64.$dt"

git add -A
git commit -m "build usd-swap-mt.$dt"
git push origin master

git tag -d "usd-swap-mt.$dt"
git tag "usd-swap-mt.$dt"
git push origin "usd-swap-mt.$dt" --force

echo ""
echo "vcarm02"
rsync -avx --progress "./dist/usd-swap-mt.arm64.$dt" vcarm02:/usr/local/bin/

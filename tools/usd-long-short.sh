#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-long-short/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-long-short.arm64.$dt" ./applications/usd-long-short
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-long-short.amd64.$dt" ./applications/usd-long-short

git add -A
git commit -m "build usd-long-short.$dt"
git push origin master
git tag -d "usd-long-short.$dt"
git tag "usd-long-short.$dt"
git push origin "usd-long-short.$dt" --force

chmod 755 "./dist/usd-long-short.amd64.$dt"

echo "hk05"
rsync -avx --progress "./dist/usd-long-short.amd64.$dt" hk05:/usr/local/bin/



#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-md-mt-q/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-md-mt-q.arm64.$dt" ./applications/usd-md-mt-q
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-md-mt-q.amd64.$dt" ./applications/usd-md-mt-q

chmod 755 "./dist/usd-md-mt-q.amd64.$dt"
chmod 755 "./dist/usd-md-mt-q.arm64.$dt"

git add -A
git commit -m "build usd-md-mt-q.$dt"
git push origin master

git tag -d "usd-md-mt-q.$dt"
git tag "usd-md-mt-q.$dt"
git push origin "usd-md-mt-q.$dt" --force

echo ""
echo "hk05"
rsync -avx --progress "./dist/usd-md-mt-q.amd64.$dt" hk05:/usr/local/bin/


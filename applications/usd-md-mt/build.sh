#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-md-mt/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-md-mt.arm64.$dt" ./applications/usd-md-mt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-md-mt.amd64.$dt" ./applications/usd-md-mt

git add -A
git commit -m "build usd-md-mt.$dt"
git push origin master
git tag -d "usd-md-mt.$dt"
git tag "usd-md-mt.$dt"
git push origin "usd-md-mt.$dt" --force

chmod 755 "./dist/usd-md-mt.amd64.$dt"
chmod 755 "./dist/usd-md-mt.arm64.$dt"

echo "" && echo "" && echo "arm1"
rsync -avx --progress "./dist/usd-md-mt.amd64.$dt" arm1:/root/exchange/

echo "" && echo "" && echo "vc01"
ssh arm1 "rsync -avx --progress /root/exchange/usd-md-mt.amd64.$dt vc01:/usr/local/bin/"

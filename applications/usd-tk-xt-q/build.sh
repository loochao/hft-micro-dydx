#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-xt-q/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-xt-q.arm64.$dt" ./applications/usd-tk-xt-q
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-xt-q.amd64.$dt" ./applications/usd-tk-xt-q

git add -A
git commit -m "build usd-tk-xt-q.$dt"
git push origin master
git tag -d "usd-tk-xt-q.$dt"
git tag "usd-tk-xt-q.$dt"
git push origin "usd-tk-xt-q.$dt" --force

chmod 755 "./dist/usd-tk-xt-q.arm64.$dt"

echo "" && echo "" && echo "arm4"
rsync -avx --progress "./dist/usd-tk-xt-q.arm64.$dt" arm4:/usr/local/bin/



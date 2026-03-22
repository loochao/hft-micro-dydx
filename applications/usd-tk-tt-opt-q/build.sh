#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt-opt-q/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt-opt-q.arm64.$dt" ./applications/usd-tk-tt-opt-q
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt-opt-q.amd64.$dt" ./applications/usd-tk-tt-opt-q

git add -A
git commit -m "build usd-tk-tt-opt-q.$dt"
git push origin master
git tag -d "usd-tk-tt-opt-q.$dt"
git tag "usd-tk-tt-opt-q.$dt"
git push origin "usd-tk-tt-opt-q.$dt" --force

chmod 755 "./dist/usd-tk-tt-opt-q.amd64.$dt"

echo "hk01"
rsync -avx --progress "./dist/usd-tk-tt-opt-q.amd64.$dt" hk01:/usr/local/bin/
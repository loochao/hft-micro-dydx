#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-rebalance-positions/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-rebalance-positions.arm64.$dt" ./applications/usd-rebalance-positions
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-rebalance-positions.amd64.$dt" ./applications/usd-rebalance-positions

git add -A
git commit -m "build usd-rebalance-positions.$dt"
git push origin master
git tag -d "usd-rebalance-positions.$dt"
git tag "usd-rebalance-positions.$dt"
git push origin "usd-rebalance-positions.$dt" --force

chmod 755 "./dist/usd-rebalance-positions.amd64.$dt"
chmod 755 "./dist/usd-rebalance-positions.arm64.$dt"



#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnus-ohlcv.$dt" ./recorders/bnus-ohlcv

git add -A
git commit -m "build bnus-ohlcv.$dt"
git push origin master

chmod 755 "./dist/bnus-ohlcv.$dt"

echo "hkhr"
rsync -avx --progress "./dist/bnus-ohlcv.$dt" hkhr:/usr/local/bin/


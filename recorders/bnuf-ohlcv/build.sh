#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnuf-ohlcv.$dt" ./recorders/bnuf-ohlcv

git add -A
git commit -m "build bnuf-ohlcv.$dt"
git push origin master

chmod 755 "./dist/bnuf-ohlcv.$dt"

echo "hkhr"
rsync -avx --progress "./dist/bnuf-ohlcv.$dt" hkhr:/usr/local/bin/


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/dxuf-bnuf-depth-and-ticker.$dt" ./recorders/dxuf-bnuf-depth-and-ticker

git add -A
git commit -m "build dxuf-bnuf-depth-and-ticker.$dt"
git push origin master

chmod 755 "./dist/dxuf-bnuf-depth-and-ticker.$dt"

echo "hk06"
rsync -avx --progress "./dist/dxuf-bnuf-depth-and-ticker.$dt" hk06:/usr/local/bin/


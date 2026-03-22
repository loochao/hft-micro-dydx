#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnts-bnuf-depth-and-ticker.$dt" ./recorders/bnts-bnuf-depth-and-ticker

git add -A
git commit -m "build bnts-bnuf-depth-and-ticker.$dt"
git push origin master

chmod 755 "./dist/bnts-bnuf-depth-and-ticker.$dt"

echo "hk08"
rsync -avx --progress "./dist/bnts-bnuf-depth-and-ticker.$dt" hk08:/usr/local/bin/


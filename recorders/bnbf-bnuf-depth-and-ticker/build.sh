#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnbf-bnuf-depth-and-ticker.$dt" ./recorders/bnbf-bnuf-depth-and-ticker

git add -A
git commit -m "build bnbf-bnuf-depth-and-ticker.$dt"
git push origin master

chmod 755 "./dist/bnbf-bnuf-depth-and-ticker.$dt"

echo "hk01"
rsync -avx --progress "./dist/bnbf-bnuf-depth-and-ticker.$dt" hk03:/usr/local/bin/


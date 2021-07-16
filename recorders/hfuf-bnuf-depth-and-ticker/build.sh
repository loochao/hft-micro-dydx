#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hfuf-bnuf-depth5-and-ticker.$dt" ./recorders/hfuf-bnuf-depth5-and-ticker

git add -A
git commit -m "build hfuf-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/hfuf-bnuf-depth5-and-ticker.$dt"

echo "hk01"
rsync -avx --progress "./dist/hfuf-bnuf-depth5-and-ticker.$dt" hk03:/usr/local/bin/


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcuf-bnuf-depth5-and-ticker.$dt" ./recorders/kcuf-bnuf-depth5-and-ticker

git add -A
git commit -m "build kcuf-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/kcuf-bnuf-depth5-and-ticker.$dt"

echo "hk01"
rsync -avx --progress "./dist/kcuf-bnuf-depth5-and-ticker.$dt" hk01:/usr/local/bin/


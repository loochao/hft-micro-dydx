#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcuf-bnus-depth5-and-ticker.$dt" ./recorders/kcuf-bnus-depth5-and-ticker

git add -A
git commit -m "build kcuf-bnus-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/kcuf-bnus-depth5-and-ticker.$dt"

echo "hk07"
rsync -avx --progress "./dist/kcuf-bnus-depth5-and-ticker.$dt" hk07:/usr/local/bin/


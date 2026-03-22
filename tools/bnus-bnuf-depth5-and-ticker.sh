#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnus-bnuf-depth5-and-ticker.$dt" ./recorders/bnus-bnuf-depth5-and-ticker

git add -A
git commit -m "build bnus-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/bnus-bnuf-depth5-and-ticker.$dt"

echo "hk04"
rsync -avx --progress "./dist/bnus-bnuf-depth5-and-ticker.$dt" hk04:/usr/local/bin/


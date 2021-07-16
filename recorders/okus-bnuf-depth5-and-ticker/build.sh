#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/okus-bnuf-depth5-and-ticker.$dt" ./recorders/okus-bnuf-depth5-and-ticker

git add -A
git commit -m "build okus-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/okus-bnuf-depth5-and-ticker.$dt"

echo "hk05"
rsync -avx --progress "./dist/okus-bnuf-depth5-and-ticker.$dt" hk04:/usr/local/bin/


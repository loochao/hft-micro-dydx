#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnbs-bnuf-depth5-and-ticker.$dt" ./recorders/bnbs-bnuf-depth5-and-ticker

git add -A
git commit -m "build bnbs-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/bnbs-bnuf-depth5-and-ticker.$dt"

echo "hk02"
rsync -avx --progress "./dist/bnbs-bnuf-depth5-and-ticker.$dt" hk02:/usr/local/bin/


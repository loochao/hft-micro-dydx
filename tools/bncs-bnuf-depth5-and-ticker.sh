#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bncs-bnuf-depth5-and-ticker.$dt" ./recorders/bncs-bnuf-depth5-and-ticker

git add -A
git commit -m "build bncs-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/bncs-bnuf-depth5-and-ticker.$dt"

echo "hk02"
rsync -avx --progress "./dist/bncs-bnuf-depth5-and-ticker.$dt" hk02:/usr/local/bin/


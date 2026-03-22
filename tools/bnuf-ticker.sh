#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnuf-ticker.$dt" ./recorders/bnuf-ticker

git add -A
git commit -m "build bnuf-ticker.$dt"
git push origin master

chmod 755 "./dist/bnuf-ticker.$dt"

echo "hk03"
rsync -avx --progress "./dist/bnuf-ticker.$dt" hk03:/usr/local/bin/


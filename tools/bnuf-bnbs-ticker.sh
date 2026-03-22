#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnuf-bnbs-ticker.$dt" ./recorders/bnuf-bnbs-ticker

git add -A
git commit -m "build bnuf-bnbs-ticker.$dt"
git push origin master

chmod 755 "./dist/bnuf-bnbs-ticker.$dt"

echo "hk04"
rsync -avx --progress "./dist/bnuf-bnbs-ticker.$dt" hk04:/usr/local/bin/


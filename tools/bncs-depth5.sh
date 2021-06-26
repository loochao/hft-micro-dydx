#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bncs-depth5.$dt" ./recorders/bncs-depth5

git add -A
git commit -m "build bncs-depth5.$dt"
git push origin master

chmod 755 "./dist/bncs-depth5.$dt"

echo "hk02"
rsync -avx --progress "./dist/bncs-depth5.$dt" hk02:/usr/local/bin/


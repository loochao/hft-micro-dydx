#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnuf.$dt" ./recorders/bnuf

git add -A
git commit -m "build bnuf.$dt"
git push origin master

chmod 755 "./dist/bnuf.$dt"

echo "hk01"
rsync -avx --progress "./dist/bnuf.$dt" hk01:/usr/local/bin/


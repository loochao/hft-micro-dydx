#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnuf-kcuf-depth5.$dt" ./recorders/bnuf-kcuf-depth5

git add -A
git commit -m "build bnuf-kcuf-depth5.$dt"
git push origin master

chmod 755 "./dist/bnuf-kcuf-depth5.$dt"

echo "hk03"
rsync -avx --progress "./dist/bnuf-kcuf-depth5.$dt" hk03:/usr/local/bin/


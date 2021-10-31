#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcuf.$dt" ./recorders/kcuf

git add -A
git commit -m "build kcuf.$dt"
git push origin master

chmod 755 "./dist/kcuf.$dt"

echo "hk02"
rsync -avx --progress "./dist/kcuf.$dt" hk02:/usr/local/bin/


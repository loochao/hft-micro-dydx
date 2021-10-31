#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcus.$dt" ./recorders/kcus

git add -A
git commit -m "build kcus.$dt"
git push origin master

chmod 755 "./dist/kcus.$dt"

echo "hk02"
rsync -avx --progress "./dist/kcus.$dt" hk02:/usr/local/bin/


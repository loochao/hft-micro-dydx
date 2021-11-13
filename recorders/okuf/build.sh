#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/okuf.$dt" ./recorders/okuf

git add -A
git commit -m "build okuf.$dt"
git push origin master

chmod 755 "./dist/okuf.$dt"

echo "hk12"
rsync -avx --progress "./dist/okuf.$dt" hk12:/usr/local/bin/


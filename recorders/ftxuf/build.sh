#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxuf.$dt" ./recorders/ftxuf

git add -A
git commit -m "build ftxuf.$dt"
git push origin master

chmod 755 "./dist/ftxuf.$dt"

echo "hk12"
rsync -avx --progress "./dist/ftxuf.$dt" hk12:/usr/local/bin/


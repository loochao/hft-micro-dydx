#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnus.$dt" ./recorders/bnus

git add -A
git commit -m "build bnus.$dt"
git push origin master

chmod 755 "./dist/bnus.$dt"

echo "hk08"
rsync -avx --progress "./dist/bnus.$dt" hk08:/usr/local/bin/


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/okus.$dt" ./recorders/okus

git add -A
git commit -m "build okus.$dt"
git push origin master

chmod 755 "./dist/okus.$dt"

echo "naeo"
rsync -avx --progress "./dist/okus.$dt" naeo:/usr/local/bin/


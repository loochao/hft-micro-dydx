#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bbuf-depth25.$dt" ./recorders/bbuf-depth25

git add -A
git commit -m "build bbuf-depth25.$dt"
git push origin master

chmod 755 "./dist/bbuf-depth25.$dt"

echo "hk06"
rsync -avx --progress "./dist/bbuf-depth25.$dt" hk06:/usr/local/bin/


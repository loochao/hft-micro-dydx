#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/okus-ticker.$dt" ./recorders/okus-ticker

git add -A
git commit -m "build okus-ticker.$dt"
git push origin master

chmod 755 "./dist/okus-ticker.$dt"

echo "hk04"
rsync -avx --progress "./dist/okus-ticker.$dt" hk04:/usr/local/bin/


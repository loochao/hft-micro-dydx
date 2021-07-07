#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/okuf-ticker.$dt" ./recorders/okuf-ticker

git add -A
git commit -m "build okuf-ticker.$dt"
git push origin master

chmod 755 "./dist/okuf-ticker.$dt"

echo "hk04"
rsync -avx --progress "./dist/okuf-ticker.$dt" hk04:/usr/local/bin/


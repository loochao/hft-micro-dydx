#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxuf-ticker.$dt" ./recorders/ftxuf-ticker

git add -A
git commit -m "build ftxuf-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxuf-ticker.$dt"

echo "hk02"
rsync -avx --progress "./dist/ftxuf-ticker.$dt" hk02:/usr/local/bin/


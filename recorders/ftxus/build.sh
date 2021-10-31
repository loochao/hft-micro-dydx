#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxus.$dt" ./recorders/ftxus

git add -A
git commit -m "build ftxus.$dt"
git push origin master

chmod 755 "./dist/ftxus.$dt"

echo "hk02"
rsync -avx --progress "./dist/ftxus.$dt" hk02:/usr/local/bin/


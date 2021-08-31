#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxus-ftxuf-ticker.$dt" ./recorders/ftxus-ftxuf-ticker

git add -A
git commit -m "build ftxus-ftxuf-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxus-ftxuf-ticker.$dt"

echo "hk11"
rsync -avx --progress "./dist/ftxus-ftxuf-ticker.$dt" hk11:/usr/local/bin/


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxuf-bnus-ticker.$dt" ./recorders/ftxuf-bnus-ticker

git add -A
git commit -m "build ftxuf-bnus-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxuf-bnus-ticker.$dt"

echo "hk11"
rsync -avx --progress "./dist/ftxuf-bnus-ticker.$dt" hk11:/usr/local/bin/


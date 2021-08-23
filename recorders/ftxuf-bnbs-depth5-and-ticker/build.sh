#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxuf-bnbs-depth5-and-ticker.$dt" ./recorders/ftxuf-bnbs-depth5-and-depth5-and-ticker

git add -A
git commit -m "build ftxuf-bnbs-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxuf-bnbs-depth5-and-ticker.$dt"

echo "hk11"
rsync -avx --progress "./dist/ftxuf-bnbs-depth5-and-ticker.$dt" hk11:/usr/local/bin/


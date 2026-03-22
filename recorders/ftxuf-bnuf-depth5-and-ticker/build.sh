#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxuf-bnuf-depth5-and-ticker.$dt" ./recorders/ftxuf-bnuf-depth5-and-ticker

git add -A
git commit -m "build ftxuf-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxuf-bnuf-depth5-and-ticker.$dt"

echo "hk10"
rsync -avx --progress "./dist/ftxuf-bnuf-depth5-and-ticker.$dt" hk10:/usr/local/bin/


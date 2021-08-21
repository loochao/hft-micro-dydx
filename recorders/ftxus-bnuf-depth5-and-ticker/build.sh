#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/ftxus-bnuf-depth5-and-ticker.$dt" ./recorders/ftxus-bnuf-depth5-and-ticker

git add -A
git commit -m "build ftxus-bnuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/ftxus-bnuf-depth5-and-ticker.$dt"

echo "hk09"
rsync -avx --progress "./dist/ftxus-bnuf-depth5-and-ticker.$dt" hk09:/usr/local/bin/


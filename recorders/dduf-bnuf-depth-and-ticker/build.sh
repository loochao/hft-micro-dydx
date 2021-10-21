#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/dduf-bnuf-depth-and-ticker.$dt" ./recorders/dduf-bnuf-depth-and-ticker

git add -A
git commit -m "build dduf-bnuf-depth-and-ticker.$dt"
git push origin master

chmod 755 "./dist/dduf-bnuf-depth-and-ticker.$dt"

echo "hk06"
rsync -avx --progress "./dist/dduf-bnuf-depth-and-ticker.$dt" hk06:/usr/local/bin/


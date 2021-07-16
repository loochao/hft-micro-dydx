#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/hbuf-bnuf-depth-and-ticker.$dt" ./recorders/hbuf-bnuf-depth-and-ticker

git add -A
git commit -m "build hbuf-bnuf-depth-and-ticker.$dt"
git push origin master

chmod 755 "./dist/hbuf-bnuf-depth-and-ticker.$dt"

echo "hk01"
rsync -avx --progress "./dist/hbuf-bnuf-depth-and-ticker.$dt" hk03:/usr/local/bin/


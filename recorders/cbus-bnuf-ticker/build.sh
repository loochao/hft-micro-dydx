#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/cbus-bnuf-ticker.$dt" ./recorders/cbus-bnuf-ticker

git add -A
git commit -m "build cbus-bnuf-ticker.$dt"
git push origin master

chmod 755 "./dist/cbus-bnuf-ticker.$dt"

echo "hk08"
rsync -avx --progress "./dist/cbus-bnuf-ticker.$dt" hk08:/usr/local/bin/


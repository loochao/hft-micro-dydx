#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcus-bnus-depth5-and-ticker.$dt" ./recorders/kcus-bnus-depth5-and-ticker

git add -A
git commit -m "build kcus-bnus-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/kcus-bnus-depth5-and-ticker.$dt"

echo "hk01"
rsync -avx --progress "./dist/kcus-bnus-depth5-and-ticker.$dt" hk01:/usr/local/bin/

echo "hk05"
ssh hk01 "rsync -avx /usr/local/bin/kcus-bnus-depth5-and-ticker.$dt hk05:hk01:/usr/local/bin/"


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcus-kcuf-depth5-and-ticker.$dt" ./recorders/kcus-kcuf-quantiles

git add -A
git commit -m "build kcus-kcuf-depth5-and-ticker.$dt"
git push origin master

chmod 755 "./dist/kcus-kcuf-depth5-and-ticker.$dt"

echo "hk08"
rsync -avx --progress "./dist/kcus-kcuf-depth5-and-ticker.$dt" hk08:/usr/local/bin/


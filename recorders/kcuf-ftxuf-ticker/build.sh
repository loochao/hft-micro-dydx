#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/kcuf-ftxuf-ticker.$dt" ./recorders/kcuf-ftxuf-ticker

git add -A
git commit -m "build kcuf-ftxuf-ticker.$dt"
git push origin master

chmod 755 "./dist/kcuf-ftxuf-ticker.$dt"

echo "hk03"
rsync -avx --progress "./dist/kcuf-ftxuf-ticker.$dt" hk03:/usr/local/bin/


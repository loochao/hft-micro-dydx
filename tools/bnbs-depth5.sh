#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnbs-depth5.$dt" ./recorders/bnbs-depth5
chmod 755 "./dist/bnbs-depth5.$dt"

git add -A
git commit -m "build bnbs-depth5.$dt"
git push origin master

git tag -d "bnbs-depth5.$dt"
git tag "bnbs-depth5.$dt"
git push origin "bnbs-depth5.$dt" --force


echo "hk02"
rsync -avx --progress "./dist/bnbs-depth5.$dt" hk02:/usr/local/bin/


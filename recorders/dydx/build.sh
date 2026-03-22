#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"

env GOOS=linux GOARCH=amd64 go build -o "./dist/dydx.$dt" ./recorders/dydx

git add -A
git commit -m "build dydx.$dt"
git push origin master

chmod 755 "./dist/dydx.$dt"

echo "naeo"
rsync -avx --progress "./dist/dydx.$dt" naeo:/usr/local/bin/


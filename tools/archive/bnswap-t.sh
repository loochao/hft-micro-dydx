#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)

env GOOS=linux GOARCH=amd64 go build -o "./dist/bnswap-t.$dt" ./applications/bnswap-t

git add -A
git commit -m "build bnswap-t.$dt"
git push origin master

chmod 755 "./dist/bnswap-t.$dt"

echo "ff04"
rsync -avx --progress "./dist/bnswap-t.$dt" ff04:/usr/local/bin/
echo "pd02"
rsync -avx --progress "./dist/bnswap-t.$dt" pd02:/usr/local/bin/

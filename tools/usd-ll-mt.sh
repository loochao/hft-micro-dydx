#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-ll-mt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-ll-mt.arm64.$dt" ./applications/usd-ll-mt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-ll-mt.amd64.$dt" ./applications/usd-ll-mt

git add -A
git commit -m "build usd-ll-mt.$dt"
git push origin master

chmod 755 "./dist/usd-ll-mt.amd64.$dt"

echo "arm1"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" arm1:/usr/local/bin/

echo "vc001"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" vc001:/usr/local/bin/

echo "xf"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" xf:/usr/local/bin/

echo "ff04"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" ff04:/usr/local/bin/

echo "luchao"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" luchao:/usr/local/bin/

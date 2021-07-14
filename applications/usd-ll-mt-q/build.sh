#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-ll-mt-q/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-ll-mt-q.arm64.$dt" ./applications/usd-ll-mt-q
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-ll-mt-q.amd64.$dt" ./applications/usd-ll-mt-q

chmod 755 "./dist/usd-ll-mt-q.amd64.$dt"
chmod 755 "./dist/usd-ll-mt-q.arm64.$dt"

git add -A
git commit -m "build usd-ll-mt-q.$dt"
git push origin master

git tag -d "usd-ll-mt-q.$dt"
git tag "usd-ll-mt-q.$dt"
git push origin "usd-ll-mt-q.$dt" --force

echo ""
echo "vcarm02"
rsync -avx --progress "./dist/usd-ll-mt-q.arm64.$dt" vcarm02:/usr/local/bin/

echo ""
echo "arm1"
rsync -avx --progress "./dist/usd-ll-mt-q.arm64.$dt" arm1:/usr/local/bin/

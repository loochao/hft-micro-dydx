#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-mt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-mt.arm64.$dt" ./applications/usd-tk-mt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-mt.amd64.$dt" ./applications/usd-tk-mt

git add -A
git commit -m "build usd-tk-mt.$dt"
git push origin master
git tag -d "usd-tk-mt.$dt"
git tag "usd-tk-mt.$dt"
git push origin "usd-tk-mt.$dt" --force

chmod 755 "./dist/usd-tk-mt.amd64.$dt"

echo "vc001"
rsync -avx --progress "./dist/usd-tk-mt.amd64.$dt" vc001:/usr/local/bin/

echo "hk05"
rsync -avx --progress "./dist/usd-tk-mt.amd64.$dt" hk05:/usr/local/bin/



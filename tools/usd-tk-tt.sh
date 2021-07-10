#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt.arm64.$dt" ./applications/usd-tk-tt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt.amd64.$dt" ./applications/usd-tk-tt

git add -A
git commit -m "build usd-tk-tt.$dt"
git push origin master
git tag -d "usd-tk-tt.$dt"
git tag "usd-tk-tt.$dt"
git push origin "usd-tk-tt.$dt" --force

chmod 755 "./dist/usd-tk-tt.amd64.$dt"

echo "hk05"
rsync -avx --progress "./dist/usd-tk-tt.amd64.$dt" hk05:/usr/local/bin/

echo "arm1"
rsync -avx --progress "./dist/usd-tk-tt.arm64.$dt" arm1:/usr/local/bin/





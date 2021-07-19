#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt-q-t/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt-q-t.arm64.$dt" ./applications/usd-tk-tt-q-t
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt-q-t.amd64.$dt" ./applications/usd-tk-tt-q-t

git add -A
git commit -m "build usd-tk-tt-q-t.$dt"
git push origin master
git tag -d "usd-tk-tt-q-t.$dt"
git tag "usd-tk-tt-q-t.$dt"
git push origin "usd-tk-tt-q-t.$dt" --force

chmod 755 "./dist/usd-tk-tt-q-t.amd64.$dt"

echo "hk07"
rsync -avx --progress "./dist/usd-tk-tt-q-t.amd64.$dt" hk07:/usr/local/bin/

echo "hk06"
ssh hk07 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q-t.amd64.$dt hk06:/usr/local/bin/"


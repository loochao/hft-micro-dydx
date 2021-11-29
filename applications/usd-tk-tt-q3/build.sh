#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt-q3/init.go

git add -A
git commit -m "build usd-tk-tt-q3.$dt"
git push origin master
git tag -d "usd-tk-tt-q3.$dt"
git tag "usd-tk-tt-q3.$dt"
git push origin "usd-tk-tt-q3.$dt" --force

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt-q3.arm64.$dt" ./applications/usd-tk-tt-q3
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt-q3.amd64.$dt" ./applications/usd-tk-tt-q3

chmod 755 "./dist/usd-tk-tt-q3.amd64.$dt"
chmod 755 "./dist/usd-tk-tt-q3.arm64.$dt"

rsync -avx --progress "./dist/usd-tk-tt-q3.arm64.$dt" tkc1:/usr/local/bin/
rsync -avx --progress "./dist/usd-tk-tt-q3.arm64.$dt" tka2:/usr/local/bin/

rsync -avx --progress "./dist/usd-tk-tt-q3.arm64.$dt" way:/usr/local/bin/
ssh way "rsync -avx --progress /usr/local/bin/usd-tk-tt-q3.arm64.$dt nv1:/usr/local/bin/"
ssh way "rsync -avx --progress /usr/local/bin/usd-tk-tt-q3.arm64.$dt nv2:/usr/local/bin/"

#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt-q2/init.go

git add -A
git commit -m "build usd-tk-tt-q2.$dt"
git push origin master
git tag -d "usd-tk-tt-q2.$dt"
git tag "usd-tk-tt-q2.$dt"
git push origin "usd-tk-tt-q2.$dt" --force

#env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt-q2.arm64.$dt" ./applications/usd-tk-tt-q2
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt-q2.amd64.$dt" ./applications/usd-tk-tt-q2

chmod 755 "./dist/usd-tk-tt-q2.amd64.$dt"

echo "" && echo "" && echo "naeo"
rsync -avx --progress "./dist/usd-tk-tt-q2.amd64.$dt" naeo:/usr/local/bin/

echo "" && echo "" && echo "way"
rsync -avx --progress "./dist/usd-tk-tt-q2.amd64.$dt" way:/usr/local/bin/


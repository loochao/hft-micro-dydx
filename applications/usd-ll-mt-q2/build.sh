#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-ll-mt-q2/init.go

env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-ll-mt-q2.arm64.$dt" ./applications/usd-ll-mt-q2
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-ll-mt-q2.amd64.$dt" ./applications/usd-ll-mt-q2

chmod 755 "./dist/usd-ll-mt-q2.amd64.$dt"
chmod 755 "./dist/usd-ll-mt-q2.arm64.$dt"

git add -A
git commit -m "build usd-ll-mt-q2.$dt"
git push origin master

git tag -d "usd-ll-mt-q2.$dt"
git tag "usd-ll-mt-q2.$dt"
git push origin "usd-ll-mt-q2.$dt" --force

echo "" && echo "" && echo "arm1"
rsync -avx --progress "./dist/usd-ll-mt-q2.arm64.$dt" arm1:/usr/local/bin/
rsync -avx --progress "./dist/usd-ll-mt-q2.amd64.$dt" arm1:/usr/local/bin/

echo "" && echo "" && echo "vc05"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-ll-mt-q2.amd64.$dt vc05:/usr/local/bin/"

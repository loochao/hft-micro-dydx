#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./round-trip-times/dydx/init.go

git add -A
git commit -m "build rtt-dydx.$dt"
git push origin master
git tag -d "rtt-dydx.$dt"
git tag "rtt-dydx.$dt"
git push origin "rtt-dydx.$dt" --force

env GOOS=linux GOARCH=arm64 go build -o "./dist/rtt-dydx.arm64.$dt" ./round-trip-times/dydx

chmod 755 "./dist/rtt-dydx.arm64.$dt"

echo "" && echo "" && echo "way"
rsync -avx --progress "./dist/rtt-dydx.arm64.$dt" way:/usr/local/bin/

ssh way "rsync -avx --progress /usr/local/bin/rtt-dydx.arm64.$dt nv1:/usr/local/bin/"
ssh way "rsync -avx --progress /usr/local/bin/rtt-dydx.arm64.$dt nv2:/usr/local/bin/"
ssh way "rsync -avx --progress /usr/local/bin/rtt-dydx.arm64.$dt arm1:/usr/local/bin/"


#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./round-trip-times/bnus/init.go

git add -A
git commit -m "build rtt-bnus.$dt"
git push origin master
git tag -d "rtt-bnus.$dt"
git tag "rtt-bnus.$dt"
git push origin "rtt-bnus.$dt" --force

env GOOS=linux GOARCH=arm64 go build -o "./dist/rtt-bnus.arm64.$dt" ./round-trip-times/bnus
env GOOS=linux GOARCH=amd64 go build -o "./dist/rtt-bnus.amd64.$dt" ./round-trip-times/bnus

chmod 755 "./dist/rtt-bnus.arm64.$dt"

#rsync -avx --progress "./dist/rtt-bnus.arm64.$dt" tka1:/usr/local/bin/
#rsync -avx --progress "./dist/rtt-bnus.amd64.$dt" tka2:/usr/local/bin/
rsync -avx --progress "./dist/rtt-bnus.arm64.$dt" tka3:/usr/local/bin/
#rsync -avx --progress "./dist/rtt-bnus.arm64.$dt" way:/usr/local/bin/
#rsync -avx --progress "./dist/rtt-bnus.amd64.$dt" way:/usr/local/bin/
#
#ssh way "rsync -avx --progress /usr/local/bin/rtt-bnus.arm64.$dt nv1:/usr/local/bin/"
#ssh way "rsync -avx --progress /usr/local/bin/rtt-bnus.arm64.$dt nv2:/usr/local/bin/"
#ssh way "rsync -avx --progress /usr/local/bin/rtt-bnus.arm64.$dt arm1:/usr/local/bin/"
#

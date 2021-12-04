#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./round-trip-times/kcuf/init.go

git add -A
git commit -m "build rtt-kcuf.$dt"
git push origin master
git tag -d "rtt-kcuf.$dt"
git tag "rtt-kcuf.$dt"
git push origin "rtt-kcuf.$dt" --force

env GOOS=linux GOARCH=arm64 go build -o "./dist/rtt-kcuf.arm64.$dt" ./round-trip-times/kcuf
env GOOS=linux GOARCH=amd64 go build -o "./dist/rtt-kcuf.amd64.$dt" ./round-trip-times/kcuf

chmod 755 "./dist/rtt-kcuf.arm64.$dt"
chmod 755 "./dist/rtt-kcuf.amd64.$dt"

rsync -avx --progress "./dist/rtt-kcuf.amd64.$dt" loochao:~/
#rsync -avx --progress "./dist/rtt-kcuf.arm64.$dt" tkc1:/usr/local/bin/
#rsync -avx --progress "./dist/rtt-kcuf.arm64.$dt" way:/usr/local/bin/
#rsync -avx --progress "./dist/rtt-kcuf.amd64.$dt" way:/usr/local/bin/

#ssh way "rsync -avx --progress /usr/local/bin/rtt-kcuf.arm64.$dt nv1:/usr/local/bin/"
#ssh way "rsync -avx --progress /usr/local/bin/rtt-kcuf.arm64.$dt nv2:/usr/local/bin/"
#ssh way "rsync -avx --progress /usr/local/bin/rtt-kcuf.arm64.$dt arm1:/usr/local/bin/"


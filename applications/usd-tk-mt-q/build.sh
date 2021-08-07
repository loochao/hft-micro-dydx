#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-mt-q/init.go

env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-mt-q.amd64.$dt" ./applications/usd-tk-mt-q

chmod 755 "./dist/usd-tk-mt-q.amd64.$dt"

echo "" && echo "" && echo "arm1"
rsync -avx --progress "./dist/usd-tk-mt-q.amd64.$dt" arm1:/usr/local/bin/

echo "" && echo "" && echo "vc06"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-mt-q.amd64.$dt vc06:/usr/local/bin/"


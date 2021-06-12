#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/opt-tt/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/hft-mirco-opt-tt.$dt" ./applications/opt-tt
env GOOS=linux GOARCH=arm64 go build -o "./dist/opt-tt.arm64.$dt" ./applications/opt-tt

git add -A
git commit -m "build hft-mirco-opt-tt.$dt"
git push origin master

chmod 755 "./dist/hft-mirco-opt-tt.$dt"
chmod 755 "./dist/opt-tt.arm64.$dt"

echo "wenzhe"
rsync -avx --progress "./dist/hft-mirco-opt-tt.$dt" wenzhe:/usr/local/bin/

echo "pd02"
rsync -avx --progress "./dist/hft-mirco-opt-tt.$dt" pd02:/usr/local/bin/

echo "arm1"
rsync -avx --progress "./dist/opt-tt.arm64.$dt" arm1:/usr/local/bin/

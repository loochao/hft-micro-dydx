#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usdt-opt-tt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usdt-opt-tt.arm64.$dt" ./applications/usdt-opt-tt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usdt-opt-tt.amd64.$dt" ./applications/usdt-opt-tt

git add -A
git commit -m "build usdt-opt-tt.$dt"
git push origin master

chmod 755 "./dist/usdt-opt-tt.amd64.$dt"

echo "arm1"
rsync -avx --progress "./dist/usdt-opt-tt.arm64.$dt" arm1:/usr/local/bin/

echo "vc001"
rsync -avx --progress "./dist/usdt-opt-tt.amd64.$dt" vc001:/usr/local/bin/

echo "xf"
rsync -avx --progress "./dist/usdt-opt-tt.amd64.$dt" xf:/usr/local/bin/

echo "ff04"
rsync -avx --progress "./dist/usdt-opt-tt.amd64.$dt" ff04:/usr/local/bin/

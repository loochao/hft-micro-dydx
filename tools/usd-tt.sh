#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tt.arm64.$dt" ./applications/usd-tt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tt.amd64.$dt" ./applications/usd-tt

chmod 755 "./dist/usd-tt.amd64.$dt"
chmod 755 "./dist/usd-tt.arm64.$dt"

git add -A
git commit -m "build usd-tt.$dt"
git push origin master

git tag -d "usd-tt.$dt"
git tag "usd-tt.$dt"
git push origin "usd-tt.$dt" --force

echo ""
echo "arm1"
rsync -avx --progress "./dist/usd-tt.arm64.$dt" arm1:/usr/local/bin/

echo ""
echo "arm2"
rsync -avx --progress "./dist/usd-tt.arm64.$dt" arm2:/usr/local/bin/

echo ""
echo "vcarm01"
rsync -avx --progress "./dist/usd-tt.arm64.$dt" vcarm01:/usr/local/bin/

echo ""
echo "vcarm03"
rsync -avx --progress "./dist/usd-tt.arm64.$dt" vcarm03:/usr/local/bin/

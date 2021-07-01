#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-ll-mt/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-ll-mt.arm64.$dt" ./applications/usd-ll-mt
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-ll-mt.amd64.$dt" ./applications/usd-ll-mt

chmod 755 "./dist/usd-ll-mt.amd64.$dt"
chmod 755 "./dist/usd-ll-mt.arm64.$dt"

git add -A
git commit -m "build usd-ll-mt.$dt"
git push origin master

git tag -d "usd-ll-mt.$dt"
git tag "usd-ll-mt.$dt"
git push origin "usd-ll-mt.$dt" --force


echo ""
echo "vcarm01"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" vcarm01:/usr/local/bin/

echo ""
echo "vcarm02"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" vcarm02:/usr/local/bin/

echo ""
echo "vcarm03"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" vcarm03:/usr/local/bin/

echo ""
echo "arm1"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" arm1:/usr/local/bin/

echo "arm2"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" arm2:/usr/local/bin/

echo "arm3"
rsync -avx --progress "./dist/usd-ll-mt.arm64.$dt" arm3:/usr/local/bin/

echo "vc001"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" vc001:/usr/local/bin/

echo "xf"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" xf:/usr/local/bin/

echo "ff04"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" ff04:/usr/local/bin/

echo "luchao"
rsync -avx --progress "./dist/usd-ll-mt.amd64.$dt" luchao:/usr/local/bin/

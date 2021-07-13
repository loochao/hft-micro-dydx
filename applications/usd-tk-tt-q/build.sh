#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/usd-tk-tt-q/init.go


env GOOS=linux GOARCH=arm64 go build -o "./dist/usd-tk-tt-q.arm64.$dt" ./applications/usd-tk-tt-q
env GOOS=linux GOARCH=amd64 go build -o "./dist/usd-tk-tt-q.amd64.$dt" ./applications/usd-tk-tt-q

git add -A
git commit -m "build usd-tk-tt-q.$dt"
git push origin master
git tag -d "usd-tk-tt-q.$dt"
git tag "usd-tk-tt-q.$dt"
git push origin "usd-tk-tt-q.$dt" --force

chmod 755 "./dist/usd-tk-tt-q.amd64.$dt"

echo "hk05"
rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" hk05:/usr/local/bin/

echo "vc001"
rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" vc001:/usr/local/bin/

echo "arm1"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm1:/usr/local/bin/
rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" arm1:/usr/local/bin/

echo "tk01"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt tk01:/usr/local/bin/"

echo "arm3"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm3:/usr/local/bin/

echo "vcarm02"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" vcarm02:/usr/local/bin/

echo "vcarm03"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" vcarm03:/usr/local/bin/

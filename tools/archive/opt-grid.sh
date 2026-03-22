#!/usr/bin/env bash

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/opt-grid/init.go


env GOOS=linux GOARCH=amd64 go build -o "./dist/opt-grid.linux.amd64.$dt" ./applications/opt-grid
env GOOS=darwin GOARCH=amd64 go build -o "./dist/opt-grid.darwin.amd64.$dt" ./applications/opt-grid
env GOOS=windows GOARCH=amd64 go build -o "./dist/opt-grid.windows.amd64.$dt.exe" ./applications/opt-grid

git add -A
git commit -m "build opt-grid.$dt"
git push origin master

chmod 755 "./dist/opt-grid.linux.amd64.$dt"

echo "ff05"
rsync -avx --progress "./dist/opt-grid.linux.amd64.$dt" ff05:/usr/local/bin/

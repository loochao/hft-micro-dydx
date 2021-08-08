#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/merge-accounts/init.go

env GOOS=windows GOARCH=amd64 go build -o "./dist/merge-accounts.$dt.exe" ./applications/merge-accounts

git add -A
git commit -m "build merge-accounts.$dt"
git push origin master
git tag -d "merge-accounts.$dt"
git tag "merge-accounts.$dt"
git push origin "merge-accounts.$dt" --force


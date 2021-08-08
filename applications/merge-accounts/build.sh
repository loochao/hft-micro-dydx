#!/usr/bin/env bash

cd ../../

dt=$(date -u +%Y%m%d)
version=" BUILD @ $(date -u '+%Y%m%d %H:%M:%S') "
echo "$version"
sed -i "" -E "s/####.+####/#### $version ####/g" ./applications/merged-accounts/init.go

env GOOS=windows GOARCH=amd64 go build -o "./dist/merge-accounts.$dt.exe" ./applications/merged-accounts

git add -A
git commit -m "build merged-accounts.$dt"
git push origin master
git tag -d "merged-accounts.$dt"
git tag "merged-accounts.$dt"
git push origin "merged-accounts.$dt" --force


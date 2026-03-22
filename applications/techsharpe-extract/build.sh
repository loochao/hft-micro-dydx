#!/usr/bin/env bash

cd ../../

env GOOS=linux GOARCH=amd64 go build -o "./dist/techsharpe-extract" ./applications/techsharpe-extract

chmod 755 "./dist/techsharpe-extract"

rsync -avx --progress "./dist/techsharpe-extract" lsyno:~/


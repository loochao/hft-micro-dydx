#!/usr/bin/env bash

cd ../../

env GOOS=linux GOARCH=amd64 go build -o "./dist/proxy" ./applications/proxy

chmod 755 "./dist/proxy"

rsync -avx --progress "./dist/proxy" hk01:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk02:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk03:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk04:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk05:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk06:/usr/local/bin/
rsync -avx --progress "./dist/proxy" hk07:/usr/local/bin/


#!/usr/bin/env bash

cd ../../

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

echo "hk07"
rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" hk07:/usr/local/bin/

echo "hk06"
ssh hk07 "rsync -avx --progress /usr/local/bin//usd-tk-tt-q.amd64.$dt hk06:/usr/local/bin/"

#echo "vc001"
#rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" vc001:/usr/local/bin/


echo "" && echo "" && echo "arm1"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm1:/usr/local/bin/
rsync -avx --progress "./dist/usd-tk-tt-q.amd64.$dt" arm1:/usr/local/bin/

echo "" && echo "" && echo "arm2"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm2:/usr/local/bin/

echo "" && echo "" && echo "arm4"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm4:/usr/local/bin/

echo "" && echo "" && echo "arm5"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm5:/usr/local/bin/


echo "" && echo "" && echo "tk01"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt tk01:/usr/local/bin/"

echo "" && echo "" && echo "tk02"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt tk02:/usr/local/bin/"

echo "" && echo "" && echo "tk03"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt tk03:/usr/local/bin/"

echo "" && echo "" && echo "vc01"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt vc01:/usr/local/bin/"

echo "" && echo "" && echo "vc02"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt vc02:/usr/local/bin/"

echo "" && echo "" && echo "vc03"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt vc03:/usr/local/bin/"

echo "" && echo "" && echo "vc04"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt vc04:/usr/local/bin/"

echo "" && echo "" && echo "vc05"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt vc05:/usr/local/bin/"

echo "" && echo "" && echo "hh01"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt hh01:/usr/local/bin/"

echo "" && echo "" && echo "hh02"
ssh arm1 "rsync -avx --progress /usr/local/bin/usd-tk-tt-q.amd64.$dt hh02:/usr/local/bin/"

echo "" && echo "" && echo "arm3"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm3:/usr/local/bin/

echo "" && echo "" && echo "arm4"
rsync -avx --progress "./dist/usd-tk-tt-q.arm64.$dt" arm4:/usr/local/bin/

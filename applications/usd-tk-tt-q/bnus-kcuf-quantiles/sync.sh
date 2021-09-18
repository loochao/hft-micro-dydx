

rsync -avx --progress --delete /Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/bnus-kcuf-quantiles/ arm1:/root/bnus-kcuf-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/bnus-kcuf-quantiles/ hh02:/usr/local/etc/f05-quantiles/"


rsync -avx --progress --delete /Users/chenjilin/Projects/hft-micro/applications/usd-tk-tt-q/configs/kcuf-bnbs-quantiles/ arm1:/root/kcuf-bnbs-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/kcuf-bnbs-quantiles/ tk02:/usr/local/etc/kcuf-bnbs-quantiles/"


rsync -avx --progress --delete /home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/bnbs-kcuf-quantiles/ arm1:/root/bnbs-kcuf-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/bnbs-kcuf-quantiles/ tk02:/usr/local/etc/f03-quantiles/"
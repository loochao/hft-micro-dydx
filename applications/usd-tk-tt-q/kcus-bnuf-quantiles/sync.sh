

rsync -avx --progress --delete /home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/kcus-bnuf-quantiles/ arm1:/root/kcus-bnuf-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/kcus-bnuf-quantiles/ tk01:/usr/local/etc/kcus-bnuf-quantiles/"
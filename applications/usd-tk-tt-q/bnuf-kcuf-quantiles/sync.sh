

rsync -avx --progress --delete /home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/bnuf-kcuf-quantiles/ arm1:/root/bnuf-kcuf-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/bnuf-kcuf-quantiles/ hh01:/usr/local/etc/f04-quantiles/"


rsync -avx --progress --delete /home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/kcus-kcuf-quantiles/ arm1:/root/kcus-kcuf-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/kcus-kcuf-quantiles/ vc02:/usr/local/etc/kcus-kcuf-quantiles/"
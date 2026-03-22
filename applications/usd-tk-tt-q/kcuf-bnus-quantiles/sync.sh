

rsync -avx --progress --delete /home/clu/Projects/hft-micro/applications/usd-tk-tt-q/configs/kcuf-bnus-quantiles/ arm1:/root/kcuf-bnus-quantiles/
ssh arm1 "rsync -avx --progress --delete /root/kcuf-bnus-quantiles/ hh02:/usr/local/etc/kcuf-bnus-quantiles/"
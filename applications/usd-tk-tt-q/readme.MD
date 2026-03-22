## Features
- [x] limit max order size
- [x] add funding rate silent period
- [x] cancel all when exit
- [x] add funding rate offset factor for spread
- [ ] collect order trade ratio 
- [x] ease in quart offset factor
- [x] xy orderSilent需要长一些，在能计算realised 
  spread再切换成enterSilent, 防止系统异常之后重复下单. 
  建议xOrderSilent 3m yOrderSilent 1m
- [x] 调整了midPrice的先后，保证就算数据不合理，influx也能保存数据
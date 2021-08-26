package common

import (
	"github.com/geometrybase/hft-micro/logger"
	"strconv"
)

func FormatFloat(value float64, prec int) string {
	price := strconv.FormatFloat(value, 'f', prec, 64)
	dot := -1
	for i := 0; i < len(price); i++ {
		if price[i] == '.' {
			dot = i
			break
		}
	}
	if dot > -1 {
		for i := len(price) - 1; i >= dot; i-- {
			if price[i] != '0' {
				if i == dot {
					logger.Debugf("%v %s", value, price[:i])
					return price[:i]
				} else {
					logger.Debugf("%v %s", value, price[:i+1])
					return price[:i+1]
				}
			}
		}
	}
	return price
}

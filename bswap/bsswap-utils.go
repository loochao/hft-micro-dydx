package bswap

import (
	"context"
	"github.com/geometrybase/hft-micro/logger"
	"math"
	"strings"
	"time"
)

func DepthLoop(ctx context.Context, symbol string, quoteQtyInUSDT, stepSize float64, api *API, pullInterval time.Duration, outputCh chan Depth) {
	buyQuoteParam := QuoteParam{
		QuoteAsset: "USDT",
		BaseAsset:  strings.Replace(symbol, "USDT", "", -1),
		QuoteQty:   quoteQtyInUSDT,
	}
	sellQuoteParam := QuoteParam{
		QuoteAsset: strings.Replace(symbol, "USDT", "", -1),
		BaseAsset:  "USDT",
		QuoteQty:   1,
	}
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	depth := Depth{
		Symbol: symbol,
	}
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			subCtx, _ := context.WithTimeout(ctx, time.Second*5)
			logger.Debugf("getQuote %v", buyQuoteParam)
			buyQuote, err := api.GetQuote(subCtx, buyQuoteParam)
			if err != nil {
				logger.Debugf("err %v", err)
				timer.Reset(pullInterval)
				continue
			}
			depth.BuyPrice = buyQuote.Price
			depth.BuyFee = buyQuote.Fee/buyQuoteParam.QuoteQty
			depth.BuySlippage = buyQuote.Slippage/buyQuoteParam.QuoteQty
			sellQuoteParam.QuoteQty = math.Ceil(quoteQtyInUSDT/buyQuote.Price/stepSize)*stepSize
			subCtx, _ = context.WithTimeout(ctx, time.Second*5)
			logger.Debugf("getQuote %v", sellQuoteParam)
			sellQuote, err := api.GetQuote(subCtx, sellQuoteParam)
			if err != nil {
				logger.Debugf("err %v", err)
				timer.Reset(pullInterval)
				continue
			}
			depth.SellPrice = 1.0/sellQuote.Price
			depth.SellFee = sellQuote.Fee/sellQuote.QuoteQty
			depth.SellSlippage = sellQuote.Slippage
			depth.Time = time.Now()
			logger.Debugf("output")
			select {
			case outputCh <- depth:
			default:
				logger.Debugf("outputCh <- depth failed, ch len %d", len(outputCh))
			}
			timer.Reset(pullInterval)
		}
	}

}

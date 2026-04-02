package dydx_v4_usdfuture

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/geometrybase/hft-micro/common"
	"github.com/geometrybase/hft-micro/logger"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

// OrderBridge handles order placement via Python helper script
// Phase 1: uses exec.Command to call place_order.py
// Phase 2: will use native Go Cosmos SDK TX signing
type OrderBridge struct {
	scriptPath       string
	address          string
	mnemonic         string
	subaccountNumber int
	proxy            string
}

func NewOrderBridge(address, mnemonic string, subaccountNumber int, proxy string) *OrderBridge {
	_, filename, _, _ := runtime.Caller(0)
	scriptDir := filepath.Dir(filename)
	scriptPath := filepath.Join(scriptDir, "scripts", "place_order.py")
	return &OrderBridge{
		scriptPath:       scriptPath,
		address:          address,
		mnemonic:         mnemonic,
		subaccountNumber: subaccountNumber,
		proxy:            proxy,
	}
}

type PythonOrderResult struct {
	Success bool   `json:"success"`
	OrderID string `json:"orderId"`
	Error   string `json:"error"`
}

func (ob *OrderBridge) PlaceOrder(ctx context.Context, param common.NewOrderParam) (*PythonOrderResult, error) {
	side := "BUY"
	if param.Side == common.OrderSideSell {
		side = "SELL"
	}
	orderType := "LIMIT"
	if param.Type == common.OrderTypeMarket {
		orderType = "MARKET"
	}
	timeInForce := "GTT"
	if param.TimeInForce == common.OrderTimeInForceIOC {
		timeInForce = "IOC"
	} else if param.TimeInForce == common.OrderTimeInForceFOK {
		timeInForce = "FOK"
	}

	args := []string{
		ob.scriptPath,
		"--action", "place",
		"--mnemonic", ob.mnemonic,
		"--market", param.Symbol,
		"--side", side,
		"--type", orderType,
		"--size", fmt.Sprintf("%.10f", param.Size),
		"--price", fmt.Sprintf("%.10f", param.Price),
		"--time-in-force", timeInForce,
		"--client-id", param.ClientID,
		"--subaccount-number", fmt.Sprintf("%d", ob.subaccountNumber),
	}
	if param.PostOnly {
		args = append(args, "--post-only")
	}
	if param.ReduceOnly {
		args = append(args, "--reduce-only")
	}
	if ob.proxy != "" {
		args = append(args, "--proxy", ob.proxy)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("python order bridge error: %v output: %s", err, string(output))
	}

	var result PythonOrderResult
	err = json.Unmarshal(output, &result)
	if err != nil {
		return nil, fmt.Errorf("parse python output error: %v output: %s", err, string(output))
	}
	if !result.Success {
		return nil, fmt.Errorf("order failed: %s", result.Error)
	}
	return &result, nil
}

func (ob *OrderBridge) CancelOrder(ctx context.Context, param common.CancelOrderParam) error {
	args := []string{
		ob.scriptPath,
		"--action", "cancel",
		"--mnemonic", ob.mnemonic,
		"--market", param.Symbol,
		"--client-id", param.ClientID,
		"--subaccount-number", fmt.Sprintf("%d", ob.subaccountNumber),
	}
	if ob.proxy != "" {
		args = append(args, "--proxy", ob.proxy)
	}

	cmd := exec.CommandContext(ctx, "python3", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("python cancel bridge error: %v output: %s", err, string(output))
	}

	var result PythonOrderResult
	err = json.Unmarshal(output, &result)
	if err != nil {
		return fmt.Errorf("parse python cancel output error: %v output: %s", err, string(output))
	}
	if !result.Success {
		return fmt.Errorf("cancel failed: %s", result.Error)
	}
	return nil
}

// watchOrder processes order requests for a single symbol
func (dd *DydxV4UsdFuture) watchOrder(
	ctx context.Context,
	symbol string,
	requestCh chan common.OrderRequest,
	responseCh chan common.Order,
	errorCh chan common.OrderError,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-dd.Done():
			return
		case req := <-requestCh:
			if req.New != nil {
				if req.New.Symbol != symbol {
					select {
					case errorCh <- common.OrderError{
						New:   req.New,
						Error: fmt.Errorf("symbol mismatch %s vs %s", req.New.Symbol, symbol),
					}:
					default:
					}
					continue
				}
				dd.submitOrder(ctx, *req.New, responseCh, errorCh)
			} else if req.Cancel != nil {
				dd.cancelOrder(ctx, *req.Cancel, errorCh)
			}
		}
	}
}

func (dd *DydxV4UsdFuture) submitOrder(ctx context.Context, param common.NewOrderParam, respCh chan common.Order, errCh chan common.OrderError) {
	subCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	_, err := dd.orderBridge.PlaceOrder(subCtx, param)
	if err != nil {
		logger.Debugf("submitOrder error: %v", err)
		select {
		case errCh <- common.OrderError{
			New:   &param,
			Error: err,
		}:
		default:
			logger.Debugf("errCh <- error failed, ch len %d", len(errCh))
		}
	}
	// Order confirmation will come via the WebSocket account channel
}

func (dd *DydxV4UsdFuture) cancelOrder(ctx context.Context, param common.CancelOrderParam, errCh chan common.OrderError) {
	subCtx, cancel := context.WithTimeout(ctx, time.Second*30)
	defer cancel()

	err := dd.orderBridge.CancelOrder(subCtx, param)
	if err != nil {
		logger.Debugf("cancelOrder error: %v", err)
		select {
		case errCh <- common.OrderError{
			Cancel: &param,
			Error:  err,
		}:
		default:
			logger.Debugf("errCh <- error failed, ch len %d", len(errCh))
		}
	}
}

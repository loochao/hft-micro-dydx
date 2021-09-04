package binance_usdtspot

import (
	"net/url"
	"strconv"
)

type SubAccountParams struct {
	Email string `json:"email"`
}

func (bkp *SubAccountParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("email", bkp.Email)
	return values
}

type SubAccountUniversalTransferParams struct {
	FromEmail       string  `json:"fromEmail"`
	ToEmail         string  `json:"toEmail"`
	FromAccountType string  `json:"fromAccountType"`
	ToAccountType   string  `json:"toAccountType"`
	Asset           string  `json:"asset"`
	Amount          float64 `json:"amount"`
}

func (bkp *SubAccountUniversalTransferParams) ToUrlValues() url.Values {
	values := url.Values{}
	values.Set("fromEmail", bkp.FromEmail)
	values.Set("toEmail", bkp.ToEmail)
	values.Set("fromAccountType", bkp.FromAccountType)
	values.Set("toAccountType", bkp.ToAccountType)
	values.Set("asset", bkp.Asset)
	values.Set("amount", strconv.FormatFloat(bkp.Amount, 'f', -1, 64))
	return values
}

package starkex

import (
	"encoding/json"
	"math/big"
)

type pedersenParams struct {
	License        [15]string       `json:"_license"`
	Comment        string           `json:"_comment"`
	FieldPrime     *big.Int         `json:"-"`
	FiledGen       *big.Int         `json:"-"`
	EcOrder        *big.Int         `json:"-"`
	Alpha          *big.Int         `json:"-"`
	Beta           *big.Int         `json:"-"`
	ConstantPoints [506][2]*big.Int `json:"-"`
}

func (pd *pedersenParams) UnmarshalJSON(data []byte) error {
	aux := &struct {
		FieldPrime     json.RawMessage         `json:"FIELD_PRIME"`
		FiledGen       json.RawMessage         `json:"FIELD_GEN"`
		EcOrder        json.RawMessage         `json:"EC_ORDER"`
		Alpha          json.RawMessage         `json:"ALPHA"`
		Beta           json.RawMessage         `json:"BETA"`
		ConstantPoints [506][2]json.RawMessage `json:"CONSTANT_POINTS"`
	}{}
	err := json.Unmarshal(data, aux)
	if err != nil {
		return err
	}
	pd.FieldPrime, _ = new(big.Int).SetString(string(aux.FieldPrime), 10)
	pd.FiledGen, _ = new(big.Int).SetString(string(aux.FiledGen), 10)
	pd.EcOrder, _ = new(big.Int).SetString(string(aux.EcOrder), 10)
	pd.Alpha, _ = new(big.Int).SetString(string(aux.Alpha), 10)
	pd.Beta, _ = new(big.Int).SetString(string(aux.Beta), 10)
	for i, row := range aux.ConstantPoints {
		pd.ConstantPoints[i][0], _ = new(big.Int).SetString(string(row[0]), 10)
		pd.ConstantPoints[i][1], _ = new(big.Int).SetString(string(row[1]), 10)
	}
	return nil
}

type EcPoint [2]*big.Int

type StarkwareOrder struct {
	OrderType                string   `json:"order_type"`
	AssetIdSynthetic         *big.Int `json:"asset_id_synthetic"`
	AssetIdCollateral        *big.Int `json:"asset_id_collateral"`
	AssetIdFee               *big.Int `json:"asset_id_fee"`
	QuantumsAmountSynthetic  int64    `json:"quantums_amount_synthetic"`
	QuantumsAmountCollateral int64    `json:"quantums_amount_collateral"`
	QuantumsAmountFee        int64    `json:"quantums_amount_fee"`
	IsBuyingSynthetic        bool     `json:"is_buying_synthetic"`
	PositionId               *big.Int `json:"position_id"`
	Nonce                    *big.Int `json:"nonce"`
	ExpirationEpochHours     *big.Int `json:"expiration_epoch_hours"`
}

func (so *StarkwareOrder) CalculateHash() []byte {
	var asset_id_sell, asset_id_buy, quantums_amount_sell, quantums_amount_buy *big.Int
	if so.IsBuyingSynthetic {
		asset_id_sell = so.AssetIdCollateral
		asset_id_buy = so.AssetIdSynthetic
		quantums_amount_sell = big.NewInt(so.QuantumsAmountCollateral)
		quantums_amount_buy = big.NewInt(so.QuantumsAmountSynthetic)
	} else {
		asset_id_sell = so.AssetIdSynthetic
		asset_id_buy = so.AssetIdCollateral
		quantums_amount_sell = big.NewInt(so.QuantumsAmountSynthetic)
		quantums_amount_buy = big.NewInt(so.QuantumsAmountCollateral)
	}
	part_1 := quantums_amount_sell
	part_1.Lsh(part_1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part_1.Add(part_1, quantums_amount_buy)
	part_1.Lsh(part_1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part_1.Add(part_1, big.NewInt(so.QuantumsAmountFee))
	part_1.Lsh(part_1, ORDER_FIELD_BIT_LENGTHS["nonce"])
	part_1.Add(part_1, so.Nonce)

	part_2 := big.NewInt(ORDER_PREFIX)
	for i := 0; i < 3; i++ {
		part_2.Lsh(part_2, ORDER_FIELD_BIT_LENGTHS["position_id"])
		part_2.Add(part_2, so.PositionId)
	}
	part_2.Lsh(part_2, ORDER_FIELD_BIT_LENGTHS["expiration_epoch_hours"])
	part_2.Add(part_2, so.ExpirationEpochHours)
	part_2.Lsh(part_2, ORDER_PADDING_BITS)

	return nil
}

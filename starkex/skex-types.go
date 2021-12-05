package starkex

import (
	"encoding/json"
	"fmt"
	"math"
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
	OrderType                string
	AssetIdSynthetic         *big.Int
	AssetIdCollateral        *big.Int
	AssetIdFee               *big.Int
	QuantumsAmountSynthetic  *big.Int
	QuantumsAmountCollateral *big.Int
	QuantumsAmountFee        *big.Int
	IsBuyingSynthetic        bool
	PositionId               *big.Int
	Nonce                    *big.Int
	ExpirationEpochHours     *big.Int
	hash                     *big.Int
}

func (so *StarkwareOrder) Sign(privateKeyHex string) (*big.Int, error) {

	return nil, nil
}

func (so *StarkwareOrder) GetHash() (*big.Int, error) {
	if so.hash != nil {
		return so.hash, nil
	} else {
		return so.CalculateHash()
	}
}

func (so *StarkwareOrder) CalculateHash() (*big.Int, error) {
	var asset_id_sell, asset_id_buy, quantums_amount_sell, quantums_amount_buy *big.Int
	if so.IsBuyingSynthetic {
		asset_id_sell = so.AssetIdCollateral
		asset_id_buy = so.AssetIdSynthetic
		quantums_amount_sell = so.QuantumsAmountCollateral
		quantums_amount_buy = so.QuantumsAmountSynthetic
	} else {
		asset_id_sell = so.AssetIdSynthetic
		asset_id_buy = so.AssetIdCollateral
		quantums_amount_sell = so.QuantumsAmountSynthetic
		quantums_amount_buy = so.QuantumsAmountCollateral
	}
	part_1 := quantums_amount_sell
	part_1.Lsh(part_1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part_1.Add(part_1, quantums_amount_buy)
	part_1.Lsh(part_1, ORDER_FIELD_BIT_LENGTHS["quantums_amount"])
	part_1.Add(part_1, so.QuantumsAmountFee)
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

	//fmt.Printf("part_1 %s\n", part_1)
	//fmt.Printf("part_2 %s\n", part_2)
	//fmt.Printf("asset_id_sell %s\n", asset_id_sell)
	//fmt.Printf("asset_id_buy %s\n", asset_id_buy)

	assetsHash, err := GetHash([]*big.Int{asset_id_sell, asset_id_buy})
	if err != nil {
		return nil, err
	}
	//fmt.Printf("assetsHash %s\n", assetsHash[0])
	//fmt.Printf("AssetIdFee %s\n", so.AssetIdFee)
	//fmt.Printf("asset_id_sell %s\n", asset_id_sell)
	//fmt.Printf("asset_id_buy %s\n", asset_id_buy)
	assetsHash, err = GetHash([]*big.Int{assetsHash[0], so.AssetIdFee})
	if err != nil {
		return nil, err
	}
	hash, err := GetHash([]*big.Int{assetsHash[0], part_1})
	if err != nil {
		return nil, err
	}
	hash, err = GetHash([]*big.Int{hash[0], part_2})
	return hash[0], err
}

func NewStarkwareOrder(
	networkId int,
	market string,
	side string,
	positionID int64,
	humanSize float64,
	humanPrice float64,
	limitFee float64,
	clientID string,
	expirationEpochSeconds int64,
) (order *StarkwareOrder, err error) {

	syntheticAsset, ok := SYNTHETIC_ASSET_MAP[market]
	if !ok {
		return nil, fmt.Errorf("%s no found in SYNTHETIC_ASSET_MAP", market)
	}
	order = &StarkwareOrder{}
	order.OrderType = "LIMIT_ORDER_WITH_FEES"
	order.AssetIdSynthetic = SYNTHETIC_ASSET_ID_MAP[syntheticAsset]
	order.AssetIdCollateral = COLLATERAL_ASSET_ID_BY_NETWORK_ID[networkId]
	order.AssetIdFee = COLLATERAL_ASSET_ID_BY_NETWORK_ID[networkId]
	order.IsBuyingSynthetic = side == ORDER_SIDE_BUY
	order.QuantumsAmountSynthetic, err = ToQuantumsExact(humanSize, syntheticAsset)
	if err != nil {
		return
	}
	if order.IsBuyingSynthetic {
		humanCost := humanPrice * humanSize
		order.QuantumsAmountCollateral = ToQuantumsRoundUp(humanCost, COLLATERAL_ASSET)
	} else {
		humanCost := humanPrice * humanSize
		order.QuantumsAmountCollateral = ToQuantumsRoundDown(humanCost, COLLATERAL_ASSET)
	}
	// The limitFee is a fraction, e.g. 0.01 is a 1 % fee.
	// It is always paid in the collateral asset.
	// Constrain the limit fee to six decimals of precision.
	// The final fee amount must be rounded up.
	limitFeeRounded := math.Floor(limitFee*1000000) / 1000000
	order.QuantumsAmountFee = big.NewInt(int64(math.Ceil(limitFeeRounded * float64(order.QuantumsAmountCollateral.Int64()))))
	order.PositionId = big.NewInt(positionID)
	order.Nonce = NonceFromClientId([]byte(clientID))
	order.ExpirationEpochHours = big.NewInt(int64(math.Ceil(float64(expirationEpochSeconds)/ONE_HOUR_IN_SECONDS)) + ORDER_SIGNATURE_EXPIRATION_BUFFER_HOURS)
	return
}

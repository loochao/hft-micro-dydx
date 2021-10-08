package starkex

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common/math"
	"math/big"
)

type Decimal256 struct {
	math.Decimal256
}

func (d *Decimal256) UnmarshalJSON(data []byte) error {
	return d.UnmarshalText(data)
}

func (d Decimal256) MarshalJSON() ([]byte, error) {
	return d.MarshalText()
}

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
	type pedersenParams struct {
		FieldPrime     json.RawMessage         `json:"FIELD_PRIME"`
		FiledGen       json.RawMessage         `json:"FIELD_GEN"`
		EcOrder        json.RawMessage         `json:"EC_ORDER"`
		Alpha          json.RawMessage         `json:"ALPHA"`
		Beta           json.RawMessage         `json:"BETA"`
		ConstantPoints [506][2]json.RawMessage `json:"CONSTANT_POINTS"`
	}
	return d.UnmarshalText(data)
}

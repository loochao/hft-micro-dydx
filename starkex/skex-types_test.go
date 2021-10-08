package starkex

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecimal256_UnmarshalJSON(t *testing.T) {
	d := Decimal256{}
	m := "2644890941682394074696857415419096381561354281743803087373802494123523779468"
	err := json.Unmarshal([]byte(m), &d)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, m, d.String())
}

func TestDecimal256_MarshalJSON(t *testing.T) {
	d := Decimal256{}
	m := "2644890941682394074696857415419096381561354281743803087373802494123523779468"
	err := json.Unmarshal([]byte(m), &d)
	if err != nil {
		t.Fatal(err)
	}
	m2, err := json.Marshal(d)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, m, string(m2))
}

func TestParsePedersenParams (t *testing.T) {
	pd := &pedersenParams{}
	err := json.Unmarshal([]byte(PedersenParamsJsonString), pd)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "3618502788666131213697322783095070105623107215331596699973092056135872020481", pd.FieldPrime.String())
	assert.Equal(t, "3618502788666131213697322783095070105526743751716087489154079457884512865583", pd.EcOrder.String())
	assert.Equal(t, "3141592653589793238462643383279502884197169399375105820974944592307816406665", pd.Beta.String())

	assert.Equal(t, "2089986280348253421170679821480865132823066470938446095505822317253594081284", pd.ConstantPoints[0][0].String())
	assert.Equal(t, "1713931329540660377023406109199410414810705867260802078187082345529207694986", pd.ConstantPoints[0][1].String())

	assert.Equal(t, "1254733481274108825174693797237617285863727098996450904398879255272288617861", pd.ConstantPoints[505][0].String())
	assert.Equal(t, "2644890941682394074696857415419096381561354281743803087373802494123523779468", pd.ConstantPoints[505][1].String())

}
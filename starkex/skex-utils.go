package starkex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"math/big"
)

func IsOne(i *big.Int) bool {
	bits := i.Bits()
	return len(bits) == 1 && bits[0] == 1 && i.Sign() > 0
}

func PiAsString(n int) string {
	if n > len(Pi1024) {
		panic("PiAsString n > 1024")
	}
	return Pi1024[:n]
}

func DivMod(n, m, p *big.Int) (*big.Int, error) {
	//Finds a non negative integer 0 <= x < p such that (m * x) % p == n
	a := big.NewInt(0)
	b := big.NewInt(0)
	c := big.NewInt(0)
	c.GCD(m, p, a, b)
	if !IsOne(c) {
		return nil, fmt.Errorf("c is not one")
	}
	a.Mul(n, a)
	return a.Mod(a, p), nil
}

func EcAdd(point1, point2 EcPoint, p *big.Int) (EcPoint, error) {
	//Gets two points on an elliptic curve mod p and returns their sum.
	//Assumes the points are given in affine form (x, y) and have different x coordinates.
	x := big.NewInt(0)
	y := big.NewInt(0)
	d := big.NewInt(0)
	x.Sub(point1[0], point2[0])
	d.Mod(x, p)
	if len(d.Bits()) == 0 {
		return EcPoint{}, errors.New("(point1[0] - point2[0]) %% p == 0")
	}
	y.Sub(point1[1], point1[1])
	m, err := DivMod(y, x, p)
	if err != nil {
		return EcPoint{}, err
	}
	x.Mul(m, m)
	x.Sub(x, point1[0])
	x.Sub(x, point1[1])
	x.Mod(x, p)
	d.Sub(point1[0], x)
	d.Mul(m, d)
	d.Sub(d, point1[1])
	d.Mod(d, p)
	return EcPoint{x, d}, nil
}

// ParseBig256 parses s as a 256 bit integer in decimal or hexadecimal syntax.
// Leading zeros are accepted. The empty string parses as zero.
func ParseBig256(s string) (*big.Int, bool) {
	if s == "" {
		return new(big.Int), true
	}
	var bigint *big.Int
	var ok bool
	if len(s) >= 2 && (s[:2] == "0x" || s[:2] == "0X") {
		bigint, ok = new(big.Int).SetString(s[2:], 16)
	} else {
		bigint, ok = new(big.Int).SetString(s, 10)
	}
	if ok && bigint.BitLen() > 256 {
		bigint, ok = nil, false
	}
	return bigint, ok
}

func NonceFromClientId(clientId []byte) *big.Int {
	hasher := sha256.New()
	hasher.Write(clientId)
	v, _ := new(big.Int).SetString(hex.EncodeToString(hasher.Sum(nil)), 16)
	v.Mod(v, big.NewInt(NONCE_UPPER_BOUND_EXCLUSIVE))
	return v
}

func ToQuantumsExact(humanAmount float64, asset string) (int64, error) {
	v := humanAmount * ASSET_RESOLUTION[asset]
	if v != float64(int64(v)) {
		return 0, fmt.Errorf(
			"amount %f is not a multiple of the quantum size %f",
			humanAmount,
			1/ASSET_RESOLUTION[asset],
		)
	}
	return int64(v), nil
}

func ToQuantumsRoundUp(humanAmount float64, asset string) int64 {
	return int64(math.Ceil(humanAmount * ASSET_RESOLUTION[asset]))
}

func ToQuantumsRoundDown(humanAmount float64, asset string) int64 {
	return int64(math.Floor(humanAmount * ASSET_RESOLUTION[asset]))
}

//def nonce_from_client_id(client_id):
//"""Generate a nonce deterministically from an arbitrary string."""
//message = hashlib.sha256()
//message.update(client_id.encode())  # Encode as UTF-8.
//return int(message.digest().hex(), 16) % NONCE_UPPER_BOUND_EXCLUSIVE

func GetHash(x *big.Int) (*EcPoint, error) {
	point := SHIFT_POINT


}

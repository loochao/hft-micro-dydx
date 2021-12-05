package starkex

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/rfc6979"
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
	//fmt.Printf("%s\n", n)
	//fmt.Printf("%s\n", m)
	//fmt.Printf("%s\n", p)
	//Finds a non negative integer 0 <= x < p such that (m * x) % p == n
	a := big.NewInt(0)
	b := big.NewInt(0)
	c := big.NewInt(0)
	c.GCD(a, b, m, p)
	if !IsOne(c) {
		return nil, fmt.Errorf("%s is not one", c)
	}
	a.Mul(n, a)
	return a.Mod(a, p), nil
}
func EcDouble(point [2]*big.Int, alpha, p *big.Int) (result [2]*big.Int, err error) {
	var x, y, m *big.Int
	x = new(big.Int).Mod(point[1], p)
	if len(x.Bits()) == 0 {
		err = fmt.Errorf("point[1] %% p != 0, %s %s", point[0], p)
		return
	}
	x = x.Mul(big.NewInt(3), point[0])
	x = x.Mul(x, point[0])
	y = new(big.Int).Mul(big.NewInt(2), point[1])
	m, err = DivMod(x, y, p)
	if err != nil {
		return
	}
	x = x.Mul(m, m)
	x = x.Sub(x, new(big.Int).Mul(big.NewInt(2), point[0]))
	x = x.Mod(x, p)
	y = y.Sub(point[0], x)
	y = y.Mul(m, y)
	y = y.Sub(y, point[1])
	y = y.Mod(y, p)
	result = [2]*big.Int{x, y}
	return
}

func EcMult(m *big.Int, point [2]*big.Int, alpha, p *big.Int) (result [2]*big.Int, err error) {
	if IsOne(m) {
		return point, nil
	}
	x := new(big.Int).Mod(m, big.NewInt(2))
	if len(x.Bits()) == 0 {
		x = x.Div(m, big.NewInt(2))
		var d [2]*big.Int
		d, err = EcDouble(point, alpha, p)
		if err != nil {
			return
		}
		return EcMult(x, d, alpha, p)
	}
	x = x.Sub(m, big.NewInt(1))
	result, err = EcMult(x, point, alpha, p)
	if err != nil {
		return
	}
	return EcAdd(result, point, p)
}

func EcAdd(point1, point2 [2]*big.Int, p *big.Int) ([2]*big.Int, error) {
	//Gets two points on an elliptic curve mod p and returns their sum.
	//Assumes the points are given in affine form (x, y) and have different x coordinates.
	x := big.NewInt(0)
	y := big.NewInt(0)

	x.Sub(point1[0], point2[0])
	x.Mod(x, p)
	if len(x.Bits()) == 0 {
		return EcPoint{}, errors.New("(point1[0] - point2[0]) %% p == 0")
	}
	y.Sub(point1[1], point2[1])
	m, err := DivMod(y, x, p)
	if err != nil {
		return EcPoint{}, err
	}
	x.Mul(m, m)
	x.Sub(x, point1[0])
	x.Sub(x, point2[0])
	x.Mod(x, p)
	y.Sub(point1[0], x)
	y.Mul(m, y)
	y.Sub(y, point1[1])
	y.Mod(y, p)
	return EcPoint{x, y}, nil
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

func ToQuantumsExact(humanAmount float64, asset string) (*big.Int, error) {
	v := humanAmount * ASSET_RESOLUTION[asset]
	if v != float64(int64(v)) {
		return nil, fmt.Errorf(
			"amount %f is not a multiple of the quantum size %f",
			humanAmount,
			1/ASSET_RESOLUTION[asset],
		)
	}
	return big.NewInt(int64(v)), nil
}

func ToQuantumsRoundUp(humanAmount float64, asset string) *big.Int {
	return big.NewInt(int64(math.Ceil(humanAmount * ASSET_RESOLUTION[asset])))
}

func ToQuantumsRoundDown(humanAmount float64, asset string) *big.Int {
	return big.NewInt(int64(math.Floor(humanAmount * ASSET_RESOLUTION[asset])))
}

//def nonce_from_client_id(client_id):
//"""Generate a nonce deterministically from an arbitrary string."""
//message = hashlib.sha256()
//message.update(client_id.encode())  # Encode as UTF-8.
//return int(message.digest().hex(), 16) % NONCE_UPPER_BOUND_EXCLUSIVE

func GetHash(xs []*big.Int) (point EcPoint, err error) {
	/*
		Similar to pedersen_hash but also returns the y coordinate of the resulting EC point.
			This function is used for testing.
	*/
	point = SHIFT_POINT
	xAnd := new(big.Int)
	bigOne := big.NewInt(1)
	for i, x := range xs {
		if x.Cmp(big.NewInt(0)) < 0 || x.Cmp(FIELD_PRIME) >= 0 {
			return point, fmt.Errorf("bad x range should 0 <= %s < %s", x, FIELD_PRIME)
		}
		pointList := CONSTANT_POINTS[2+i*N_ELEMENT_BITS_HASH : 2+(i+1)*N_ELEMENT_BITS_HASH]
		x = new(big.Int).Set(x)
		for _, pt := range pointList {
			if pt[0].Cmp(point[0]) == 0 {
				return point, fmt.Errorf("unhashable input %s", point[0])
			}
			if len(xAnd.And(x, bigOne).Bits()) != 0 {
				point, err = EcAdd(point, pt, FIELD_PRIME)
				if err != nil {
					return point, err
				}
			}
			x.Rsh(x, 1)

		}
		if len(x.Bits()) != 0 {
			return point, fmt.Errorf("%s should be zero", x)
		}
	}

	return point, nil
}

func GoSign(msgHash, privateKey, seed *big.Int) error {
	if msgHash.Cmp(big.NewInt(0)) < 0 || msgHash.Cmp(N_ELEMENT_BITS_ECDSA_MAX_VALUE) >= 0 {
		return fmt.Errorf("hash not signable")
	}
	for {
		k := generateSecretRfc6979(msgHash, privateKey, seed)
		if seed == nil {
			seed = big.NewInt(1)
		} else {
			seed.And(seed, big.NewInt(1))
		}
		logger.Debugf("%s\n", k)
	}
	return nil
}

func generateSecretRfc6979(msgHash, privateKey, seed *big.Int) (result *big.Int) {
	msgHashLen := msgHash.BitLen()
	msgHashLenMod8 := msgHashLen % 8
	if msgHashLenMod8 >= 1 && msgHashLenMod8 <= 4 && msgHashLen >= 248 {
		msgHash.Mul(msgHash, big.NewInt(16))
	}
	extraEntropy := make([]byte, 0)
	if seed != nil {
		extraEntropy = seed.Bytes()
	}
	rfc6979.GenerateSecret(EC_ORDER, privateKey, sha256.New, msgHash.Bytes(), extraEntropy, func(r *big.Int) bool {
		fmt.Printf("RESULT %s\n", r)
		result = r
		return true
	})
	return
}

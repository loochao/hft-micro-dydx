package starkex_test

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"github.com/geometrybase/hft-micro/starkex"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
	"time"
)

const (
	MOCK_SIGNATURE         = "00cecbe513ecdbf782cd02b2a5efb03e58d5f63d15f2b840e9bc0029af04e8dd0090b822b16f50b2120e4ea9852b340f7936ff6069d02acca02f2ed03029ace5"
	MOCK_PUBLIC_KEY_EVEN_Y = "5c749cd4c44bdc730bc90af9bfbdede9deb2c1c96c05806ce1bc1cb4fed64f7"
	MOCK_SIGNATURE_EVEN_Y  = "00fc0756522d78bef51f70e3981dc4d1e82273f59cdac6bc31c5776baabae6ec0158963bfd45d88a99fb2d6d72c9bbcf90b24c3c0ef2394ad8d05f9d3983443a"
)

var MOCK_PUBLIC_KEY, _ = new(big.Int).SetString("3b865a18323b8d147a12c556bfb1d502516c325b1477a23ba6c77af31f020fd", 16)
var MOCK_PRIVATE_KEY, _ = new(big.Int).SetString("58c7d5a90b1776bde86ebac077e053ed85b0f7164f53b080304a531947f46e3", 16)

func TestHashOrder(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		t.Fatal(err)
	}
	order, err := starkex.NewStarkwareOrder(
		starkex.NETWORK_ID_ROPSTEN,
		starkex.MARKET_ETH_USD,
		starkex.ORDER_SIDE_BUY,
		12345,
		145.0005,
		350.00067,
		0.125,
		"This is an ID that the client came up with to describe this order",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}
	hash, err := order.CalculateHash()
	if err != nil {
		t.Fatal(err)
	}
	answer, _ := new(big.Int).SetString("2399267126880666724459410666672606885138497587740885437088147399489673150280", 10)
	assert.Equal(t, 0, answer.Cmp(hash))
}

func BenchmarkHashOrder(b *testing.B) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < b.N; i++ {
		so, err := starkex.NewStarkwareOrder(
			starkex.NETWORK_ID_ROPSTEN,
			starkex.MARKET_ETH_USD,
			starkex.ORDER_SIDE_BUY,
			12345,
			145.0005,
			350.00067,
			0.125,
			"This is an ID that the client came up with to describe this order",
			tt.Unix(),
		)
		if err != nil {
			b.Fatal(err)
		}
		_, err = so.CalculateHash()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestSignOrder(t *testing.T) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		t.Fatal(err)
	}
	so, err := starkex.NewStarkwareOrder(
		starkex.NETWORK_ID_ROPSTEN,
		starkex.MARKET_ETH_USD,
		starkex.ORDER_SIDE_BUY,
		12345,
		145.0005,
		350.00067,
		0.125,
		"This is an ID that the client came up with to describe this order",
		tt.Unix(),
	)
	if err != nil {
		t.Fatal(err)
	}

	sg, err := so.Sign(MOCK_PRIVATE_KEY)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, MOCK_SIGNATURE, sg)
}

func BenchmarkSignOrder(b *testing.B) {
	tt, err := time.Parse("2006-01-02T15:04:05.999Z", "2020-09-17T04:15:55.028Z")
	if err != nil {
		b.Fatal(err)
	}
	errCount := 0.0
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		so, err := starkex.NewStarkwareOrder(
			starkex.NETWORK_ID_ROPSTEN,
			starkex.MARKET_ETH_USD,
			starkex.ORDER_SIDE_BUY,
			12345,
			145.0005,
			350.00067,
			0.125,
			"This is an ID that the client came up with to describe this order",
			tt.Unix(),
		)
		if err != nil {
			b.Fatal(err)
		}
		_, err = so.Sign(MOCK_PRIVATE_KEY)
		if err != nil {
			errCount ++
			//b.Fatal(err)
		}
	}
	logger.Debugf("%f", errCount/float64(b.N))
}

func TestGCD(t *testing.T) {
	x := big.NewInt(0)
	y := big.NewInt(0)
	c := big.NewInt(0)
	a := big.NewInt(125)
	b := big.NewInt(10000)
	c.GCD(x, y, a, b)

	fmt.Printf("%s*%s + %s*%s = %s\n", x, a, y, b, c)
}

func TestEcAdd(t *testing.T) {
	point1 := starkex.EcPoint{}
	point2 := starkex.EcPoint{}
	answer := starkex.EcPoint{}
	filedPrime := new(big.Int)
	point1[0], _ = new(big.Int).SetString("2089986280348253421170679821480865132823066470938446095505822317253594081284", 10)
	point1[1], _ = new(big.Int).SetString("1713931329540660377023406109199410414810705867260802078187082345529207694986", 10)
	point2[0], _ = new(big.Int).SetString("100775230685312048816501234355008830851785728808228209380195522984287974518", 10)
	point2[1], _ = new(big.Int).SetString("3198314560325546891798262260233968848553481119985289977998522774043088964633", 10)
	filedPrime, _ = new(big.Int).SetString("3618502788666131213697322783095070105623107215331596699973092056135872020481", 10)
	answer[0], _ = new(big.Int).SetString("1637368371864026355245122316446106576874611007407245016652355316950184561542", 10)
	answer[1], _ = new(big.Int).SetString("2972442824041547060031660375530558262127955088805062438335262544824625022241", 10)
	pt, err := starkex.EcAdd(point1, point2, filedPrime)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, pt[0].Cmp(answer[0]))
	assert.Equal(t, 0, pt[1].Cmp(answer[1]))

	point1[0], _ = new(big.Int).SetString("1637368371864026355245122316446106576874611007407245016652355316950184561542", 10)
	point1[1], _ = new(big.Int).SetString("2972442824041547060031660375530558262127955088805062438335262544824625022241", 10)
	point2[0], _ = new(big.Int).SetString("1337726844298689299569036965005062374791732295462158862097564380968412485659", 10)
	point2[1], _ = new(big.Int).SetString("3094702644796621069343809899235459280874613277076424986270525032931210979878", 10)
	filedPrime, _ = new(big.Int).SetString("3618502788666131213697322783095070105623107215331596699973092056135872020481", 10)
	answer[0], _ = new(big.Int).SetString("2466881358002133364822637278001945633159199669109451817445969730922553850042", 10)
	answer[1], _ = new(big.Int).SetString("1271130375644313270454207628605737994583176473391656197852450822562881214187", 10)
	pt, err = starkex.EcAdd(point1, point2, filedPrime)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, pt[0].Cmp(answer[0]))
	assert.Equal(t, 0, pt[1].Cmp(answer[1]))

	point1[0], _ = new(big.Int).SetString("2466881358002133364822637278001945633159199669109451817445969730922553850042", 10)
	point1[1], _ = new(big.Int).SetString("1271130375644313270454207628605737994583176473391656197852450822562881214187", 10)
	point2[0], _ = new(big.Int).SetString("855657745844414012325398643860801166203065495756352613799675558543302817038", 10)
	point2[1], _ = new(big.Int).SetString("1379036914678019505188657918379814767819231204146554192918997656166330268474", 10)
	filedPrime, _ = new(big.Int).SetString("3618502788666131213697322783095070105623107215331596699973092056135872020481", 10)
	answer[0], _ = new(big.Int).SetString("492818067953291127695451335951345651920109890520024940835615895044246976579", 10)
	answer[1], _ = new(big.Int).SetString("1149842739045414779301434642411812034512038116281457368094778245605718550168", 10)
	pt, err = starkex.EcAdd(point1, point2, filedPrime)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, pt[0].Cmp(answer[0]))
	assert.Equal(t, 0, pt[1].Cmp(answer[1]))

	point1[0], _ = new(big.Int).SetString("492818067953291127695451335951345651920109890520024940835615895044246976579", 10)
	point1[1], _ = new(big.Int).SetString("1149842739045414779301434642411812034512038116281457368094778245605718550168", 10)
	point2[0], _ = new(big.Int).SetString("2860710426779608457334569506319606721823380279653117262373857444958848532006", 10)
	point2[1], _ = new(big.Int).SetString("1390846552016301495855136360351297463700036202880431397235275981413499580322", 10)
	filedPrime, _ = new(big.Int).SetString("3618502788666131213697322783095070105623107215331596699973092056135872020481", 10)
	answer[0], _ = new(big.Int).SetString("3135801844875693822174243539459344397133928002790813799052711526459420344361", 10)
	answer[1], _ = new(big.Int).SetString("1418122846584517037160303273194013812207498515965424206485619182204740913242", 10)
	pt, err = starkex.EcAdd(point1, point2, filedPrime)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, pt[0].Cmp(answer[0]))
	assert.Equal(t, 0, pt[1].Cmp(answer[1]))
}

func TestGetHash(t *testing.T) {
	x, _ := new(big.Int).SetString("1093205074515244646656179739104081883720670447274991282058399169276196848522", 10)
	answer, _ := new(big.Int).SetString("2855274266086995320413009292647740303355258199821409804862117800451140988819", 10)
	pt, err := starkex.GetHash([]*big.Int{x})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, answer.Cmp(pt[0]))
	x1, _ := new(big.Int).SetString("1471675751746937423994835901984787733034746094072758645529066106420348927196", 10)
	x2, _ := new(big.Int).SetString("1244395526148093605117595054168172062218752879259769683800039479765231001178", 10)
	answer, _ = new(big.Int).SetString("3197185028469094583235190523237161761112590433058643901650933274684999410057", 10)
	pt, err = starkex.GetHash([]*big.Int{x1, x2})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, answer.Cmp(pt[0]))

	x1, _ = new(big.Int).SetString("3197185028469094583235190523237161761112590433058643901650933274684999410057", 10)
	x2, _ = new(big.Int).SetString("74171605843675424352885220490031857899938574853322944333809", 10)
	answer, _ = new(big.Int).SetString("1067101959365932490585603870096257964283976674639120806958374791609453687017", 10)
	pt, err = starkex.GetHash([]*big.Int{x1, x2})
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, answer.Cmp(pt[0]))
}

func TestDivMod(t *testing.T) {
	v, err := starkex.DivMod(big.NewInt(1), big.NewInt(333), big.NewInt(10000))
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, v.Cmp(big.NewInt(6997)))
	vv := new(big.Int).Mul(big.NewInt(333), v)
	assert.Equal(t, 0, vv.Mod(vv, big.NewInt(10000)).Cmp(big.NewInt(1)))
	answer, _ := new(big.Int).SetString("1008357535208024310512793551212243230301413101977305291975416549754692247557", 10)
	n := big.NewInt(10)
	m, _ := new(big.Int).SetString("1093205074515244646656179739104081883720670447274991282058399169276196848522", 10)
	p, _ := new(big.Int).SetString("3618502788666131213697322783095070105623107215331596699973092056135872020481", 10)

	v, err = starkex.DivMod(n, m, p)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, 0, answer.Cmp(v))
	vv = new(big.Int).Mul(m, v)
	assert.Equal(t, 0, vv.Mod(vv, p).Cmp(n))
}



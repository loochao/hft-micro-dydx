// The following directive is necessary to make the package coherent:

// +build ignore

// This program generates contributors.go. It can be invoked by running
// go generate
package main

import (
	"fmt"
	"github.com/geometrybase/hft-micro/logger"
	"log"
	"os"
	"sort"
	"strings"
	"text/template"
)

//BTCUSDT,LTCUSDT,ETHUSDT,NEOUSDT,QTUMUSDT,EOSUSDT,ZRXUSDT,OMGUSDT,LRCUSDT,TRXUSDT,KNCUSDT,IOTAUSDT,LINKUSDT,CVCUSDT,ETCUSDT,ZECUSDT,BATUSDT,DASHUSDT,XMRUSDT,ENJUSDT,XRPUSDT,STORJUSDT,BTSUSDT,ADAUSDT,XLMUSDT,WAVESUSDT,ICXUSDT,RLCUSDT,IOSTUSDT,BLZUSDT,ONTUSDT,ZILUSDT,ZENUSDT,THETAUSDT,VETUSDT,RENUSDT,MATICUSDT,ATOMUSDT,FTMUSDT,CHZUSDT,ALGOUSDT,DOGEUSDT,ANKRUSDT,TOMOUSDT,BANDUSDT,XTZUSDT,KAVAUSDT,BCHUSDT,SOLUSDT,HNTUSDT,COMPUSDT,MKRUSDT,SXPUSDT,SNXUSDT,DOTUSDT,RUNEUSDT,BALUSDT,YFIUSDT,SRMUSDT,CRVUSDT,SANDUSDT,OCEANUSDT,LUNAUSDT,RSRUSDT,TRBUSDT,EGLDUSDT,BZRXUSDT,KSMUSDT,SUSHIUSDT,YFIIUSDT,BELUSDT,UNIUSDT,AVAXUSDT,FLMUSDT,ALPHAUSDT,NEARUSDT,AAVEUSDT,FILUSDT,CTKUSDT,AXSUSDT,AKROUSDT,SKLUSDT,GRTUSDT,1INCHUSDT,LITUSDT,RVNUSDT,SFPUSDT,REEFUSDT,DODOUSDT,COTIUSDT,CHRUSDT,ALICEUSDT,HBARUSDT,MANAUSDT,STMXUSDT,UNFIUSDT,XEMUSDT

type Group struct {
	Symbols []string
	Depth   int
	Path    string
}

type Branch struct {
	Symbols  []string
	Depth    int
	Path     string
	Branches []*Branch
}

func HasAllLeaves(b *Branch) bool {
	if len(b.Branches) == 0 {
		return len(b.Symbols) == 1
	} else {
		for _, bb := range b.Branches {
			if !HasAllLeaves(bb) {
				return false
			}
		}
	}
	return true
}

func WalkBranch(b *Branch) {
	if len(b.Symbols) <= 1 {
		//logger.Debugf("%s %s", b.Path, b.Symbols)
		return
	}
	b.Branches = make([]*Branch, 0)
	depth := b.Depth + 1
	symbolsMap := make(map[string][]string)
	for _, symbol := range b.Symbols {
		if _, ok := symbolsMap[symbol[:depth]]; !ok {
			symbolsMap[symbol[:depth]] = []string{symbol}
		} else {
			symbolsMap[symbol[:depth]] = append(symbolsMap[symbol[:depth]], symbol)
		}
	}
	for path, symbols := range symbolsMap {
		b.Branches = append(b.Branches, &Branch{
			Path:    path,
			Symbols: symbols,
			Depth:   depth,
		})
	}
	for _, branch := range b.Branches {
		WalkBranch(branch)
	}
}

func GetDepthOffset(depth int) string {
	offset := "  "
	for i := 0; i < depth; i++ {
		offset += "  "
	}
	return offset
}

func GetBranchSwitchCode(b *Branch, symbols []string) string {
	if len(b.Branches) < 1 {
		logger.Debugf("%s %s", b.Path, b.Symbols)
		return fmt.Sprintf(
			"%scase '%s':\n%s  return %d\n",
			GetDepthOffset(b.Depth),
			b.Symbols[0][b.Depth-1:b.Depth],
			GetDepthOffset(b.Depth),
			sort.SearchStrings(symbols, b.Symbols[0]),
			//b.Symbols[0],
		)
	} else {
		cases := ""
		for _, b1 := range b.Branches {
			cases += GetBranchSwitchCode(b1, symbols)
		}
		return fmt.Sprintf(
			"%scase '%s':\n%s  switch symbol[%d]{\n%s%sdefault:\n%sreturn -1\n%s}\n",
			GetDepthOffset(b.Depth),
			b.Symbols[0][b.Depth-1:b.Depth],
			GetDepthOffset(b.Depth),
			b.Depth,
			cases,
			GetDepthOffset(b.Depth+1),
			GetDepthOffset(b.Depth+2),
			GetDepthOffset(b.Depth),
		)
	}
}

func main() {

	symbolsStr := os.Getenv("BN_SYMBOLS")
	symbols := strings.Split(symbolsStr, ",")
	sort.Strings(symbols)
	logger.Debugf("%s", symbols)

	tree := &Branch{
		Symbols: symbols,
		Depth:   0,
		Path:    "",
	}
	WalkBranch(tree)
	cases := ""
	for _, b1 := range tree.Branches {
		cases += GetBranchSwitchCode(b1, symbols)
	}
	codes := "\nfunc GetSymbolIndex(symbol string) int{\n"
	codes += fmt.Sprintf("  switch symbol[0] {\n%s  default:\n    return -1\n  }\n", cases)
	codes += "  return -1\n"
	codes += "}"
	logger.Debugf("%s", codes)

	//tree := map[int]map[string]Group{}
	//done := false
	//depth := 0
	//for !done {
	//	done = true
	//	tree[depth] = make(map[string]Group)
	//	if depth == 0 {
	//		for _, symbol := range symbols {
	//			if g, ok := tree[depth][symbol[:depth+1]]; !ok {
	//				tree[depth][symbol[:depth+1]] = Group{
	//					Symbols: []string{symbol},
	//					Path:    symbol[:depth+1],
	//					Depth:   depth,
	//				}
	//			} else {
	//				g.Symbols = append(g.Symbols, symbol)
	//				tree[depth][symbol[:depth+1]] = g
	//			}
	//		}
	//	} else {
	//		for _, g := range tree[depth-1] {
	//			if len(g.Symbols) == 1 {
	//				continue
	//			}
	//			for _, symbol := range g.Symbols {
	//				if g, ok := tree[depth][symbol[:depth+1]]; !ok {
	//					tree[depth][symbol[:depth+1]] = Group{
	//						Symbols: []string{symbol},
	//						Path:    symbol[:depth+1],
	//						Depth:   depth,
	//					}
	//				} else {
	//					g.Symbols = append(g.Symbols, symbol)
	//					tree[depth][symbol[:depth+1]] = g
	//				}
	//			}
	//		}
	//	}
	//	for _, g := range tree[depth] {
	//		if len(g.Symbols) > 1 {
	//			done = false
	//		}
	//	}
	//	depth += 1
	//	//done = true
	//}
	//code := "function GetSymbolIndex(symbol string) int {\n"
	//code += "  switch symbol[0] {\n"
	//for i := 0; i < len(tree); i++ {
	//	for _, g := range tree[i] {
	//		code += fmt.Sprintf("    case '%s':", g.Path)
	//	}
	//}
	//code += "  }\n"
	//code += "}\n"
	//
	//logger.Debugf("%s", code)

	//const url = "https://github.com/golang/go/raw/master/CONTRIBUTORS"
	//
	//rsp, err := http.Get(url)
	//die(err)
	//defer rsp.Body.Close()
	//
	//sc := bufio.NewScanner(rsp.Body)
	//carls := []string{}
	//
	//for sc.Scan() {
	//	if strings.Contains(sc.Text(), "Carl") {
	//		carls = append(carls, sc.Text())
	//	}
	//}
	//
	//die(sc.Err())
	//
	//f, err := os.Create("contributors.go")
	//die(err)
	//defer f.Close()
	//
	//packageTemplate.Execute(f, struct {
	//	Timestamp time.Time
	//	URL       string
	//	Carls     []string
	//}{
	//	Timestamp: time.Now(),
	//	URL:       url,
	//	Carls:     carls,
	//})
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var packageTemplate = template.Must(template.New("").Parse(`// Code generated by go generate; DO NOT EDIT.
// This file was generated by robots at
// {{ .Timestamp }}
// using data from
// {{ .URL }}
package project

var Contributors = []string{
{{- range .Carls }}
	{{ printf "%q" . }},
{{- end }}
}
`))

module github.com/geometrybase/hft-micro

//replace github.com/geometrybase/hft-micro/bnspot => ./bnspot
//
//replace github.com/geometrybase/hft-micro/common => ./common
//
//replace github.com/geometrybase/hft-micro/logger => ./logger

go 1.16

require (
	github.com/certifi/gocertifi v0.0.0-20200922220541-2c3bb06c6054 // indirect
	github.com/ethereum/go-ethereum v1.10.9
	github.com/go-echarts/go-echarts/v2 v2.2.4
	github.com/gorilla/websocket v1.4.2
	github.com/influxdata/influxdb1-client v0.0.0-20200827194710-b269163b24ab
	github.com/leesper/go_rng v0.0.0-20190531154944-a612b043e353
	github.com/minio/simdjson-go v0.2.2
	github.com/montanaflynn/stats v0.6.6
	github.com/stretchr/testify v1.7.0
	gonum.org/v1/gonum v0.9.0
	gopkg.in/yaml.v2 v2.4.0
	gorgonia.org/gorgonia v0.9.17
)

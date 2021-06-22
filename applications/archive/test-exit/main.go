package main

import (
	"fmt"
	"time"
)

func main() {
	date, err := time.Parse("20060102", "20210204")
	if err != nil {
		panic(err)
	}
	fmt.Printf("%v", date)
}


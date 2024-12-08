package main

import (
	"flag"
)

var flagRunAddr string
var reportInterval, pollInterval int64

func parseFlags() {
	flag.StringVar(&flagRunAddr, "a", "localhost:8080", "address and port to run server")
	flag.Int64Var(&reportInterval, "r", 10, "reportInterval period")
	flag.Int64Var(&pollInterval, "p", 2, "pollInterval period")
	flag.Parse()
}

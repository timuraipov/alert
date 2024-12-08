package main

import (
	"net/http"

	"github.com/timuraipov/alert/internal/server"
)

func main() {
	parseFlags()

	server := server.New()
	err := http.ListenAndServe(flagRunAddr, server)
	if err != nil {
		panic(err)
	}
}

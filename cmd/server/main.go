package main

import (
	"github.com/timuraipov/alert/internal/server"
)

func main() {
	if err := server.Run(); err != nil {
		panic(err)
	}
}

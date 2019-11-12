package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/genesis32/complianceweb/server"
)

func main() {
	rand.Seed(time.Now().UnixNano())
	server := server.NewServer()
	err := server.Startup()
	if err != nil {
		log.Fatal(err)
	}
	defer server.Shutdown()

	server.Serve()
}

package main

import (
	"encoding/gob"
	"log"

	"github.com/genesis32/complianceweb/server"
)

func main() {
	gob.Register(map[string]interface{}{})
	server := server.NewServer()
	err := server.Startup()
	if err != nil {
		log.Fatal(err)
	}
	defer server.Shutdown()

	server.Serve()
}

package main

import (
	"encoding/gob"

	"github.com/genesis32/complianceweb/server"
)

func main() {
	gob.Register(map[string]interface{}{})
	server := server.NewServer()
	server.Startup()
	server.Serve()
}

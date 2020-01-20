package main

import (
	"log"

	"github.com/genesis32/complianceweb/cmd"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		log.Fatalln(err)
	}
}

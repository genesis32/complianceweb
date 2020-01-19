package main

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/genesis32/complianceweb/cmd"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	if err := cmd.RootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

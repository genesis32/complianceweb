package utils

import (
	"log"
	"math/rand"
)

func GenerateRandomBytes(len int) []byte {
	arr := make([]byte, len)
	if _, err := rand.Read(arr); err != nil {
		log.Fatalf("generating random bytes: %w", err)
	}
	return arr
}

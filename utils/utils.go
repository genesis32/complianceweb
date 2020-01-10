package utils

import (
	"log"
	"math/rand"
	"strconv"
)

func StringToInt64(v string) (int64, error) {
	ret, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, err
	}
	return ret, nil
}

func GenerateRandomBytes(len int) []byte {
	arr := make([]byte, len)
	if _, err := rand.Read(arr); err != nil {
		log.Fatalf("generating random bytes: %w", err)
	}
	return arr
}

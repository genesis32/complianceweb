package utils

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"

	"github.com/dgrijalva/jwt-go"
)

type OpenIDClaims map[string]interface{}

func GetNextUniqueId() int64 {
	return rand.Int63()
}

func GenerateTestJwt(sub string) string {

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"given_name":  "John",
		"family_name": "Smith",
		"nickname":    "jsmith",
		"name":        "John Smith",
		"picture":     "https://lh3.googleusercontent.com/a-/AAuE7mCY2TSqk_4WBFHXLzi-GX_ircRYCFGwzoYMDVFF3eU",
		"locale":      "en",
		"updated_at":  "2020-01-19T01:11:51.254Z",
		"iss":         "https://issuer",
		"sub":         sub,
		"aud":         "foo",
		"iat":         1579396311,
		"exp":         1879432311,
	})

	// Just sign this thing with a blank key (all 0s)
	key := make([]byte, 64)
	ret, err := token.SignedString(key)
	if err != nil {
		log.Fatal(err)
	}
	return ret
}

func ParseTestJwt(jwtBase64 string, key []byte) OpenIDClaims {
	var err error
	token, err := jwt.Parse(jwtBase64, func(token *jwt.Token) (interface{}, error) {
		// Don't forget to validate the alg is what you expect:
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return key, nil
	})

	if err != nil {
		log.Fatal(err)
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		openIDClaims := make(OpenIDClaims)
		for k, v := range claims {
			openIDClaims[k] = v
		}
		return openIDClaims
	}
	return nil
}

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

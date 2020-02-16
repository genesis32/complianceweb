package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/utils"
)

func init() {
	if errs := os.Chdir("../"); errs != nil {
		panic(errs)
	}
}

func generateTestJwt() string {
	jwt := utils.GenerateTestJwt(fmt.Sprintf("oauth|%d", utils.GetNextUniqueId()))
	return jwt
}

func simulateLogin(handler dao.DaoHandler, inviteCode string) string {
	jwt := generateTestJwt()

	hsKey := make([]byte, 64)
	claims := utils.ParseTestJwt(jwt, hsKey)
	handler.InitUserFromInviteCode(inviteCode, claims["sub"].(string))
	return jwt
}

func createBaseRequest(t *testing.T, server *httptest.Server, bearerToken, method, path string) *http.Request {
	req, err := http.NewRequest(method, server.URL+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("Authorization", "Bearer "+bearerToken)
	return req
}

func createJsonRequest(data interface{}) io.ReadCloser {
	ret, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	return ioutil.NopCloser(bytes.NewReader(ret))
}

func addJsonBody(req *http.Request, data interface{}) {
	req.Header["Content-Type"] = append(req.Header["Content-Type"], "application/json")
	req.Body = createJsonRequest(data)
}

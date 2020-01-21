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
	"testing"

	"github.com/genesis32/complianceweb/dao"
	"github.com/genesis32/complianceweb/utils"
)

func simulateLogin(handler dao.DaoHandler, inviteCode string) string {
	jwt := utils.GenerateTestJwt(fmt.Sprintf("oauth|%d", utils.GetNextUniqueId()))

	key := make([]byte, 64)
	claims := utils.ParseTestJwt(jwt, key)
	handler.InitUserFromInviteCode(inviteCode, claims["sub"].(string))
	return jwt
}

func createBaseRequest(t *testing.T, server *httptest.Server, bearerToken, method, path string) *http.Request {
	req, err := http.NewRequest(method, server.URL+path, nil)
	req.Header.Add("Authorization", "Bearer "+bearerToken)
	if err != nil {
		t.Fatal(err)
	}
	return req
}

func createJsonRequest(data map[string]interface{}) io.ReadCloser {
	ret, err := json.Marshal(data)
	if err != nil {
		log.Fatal(err)
	}
	return ioutil.NopCloser(bytes.NewReader(ret))
}

func addJsonBody(req *http.Request, jsonMap map[string]interface{}) {
	req.Header["Content-Type"] = append(req.Header["Content-Type"], "application/json")
	req.Body = createJsonRequest(jsonMap)
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/server"
)

type genericJson map[string]interface{}

var initialUserJwtFiles = []string{"test/data/105843250540508297717.txt"}
var fixedJwts []string
var systemAdminJwt string

var baseServer *server.Server

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

func testBootstrap(server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		req := createBaseRequest(t, server, fixedJwts[0], "POST", "/system/bootstrap")
		jsonReq := make(map[string]interface{})
		jsonReq["SystemAdminName"] = "SystemAdmin0"
		req.Body = createJsonRequest(jsonReq)

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("statuscode expected: StatusOK got: %d", resp.StatusCode)
		}
		// Mimic the user logging in
		{
			var jsonResp genericJson
			json.NewDecoder(resp.Body).Decode(&jsonResp)
			v := jsonResp["InviteCode"].(string)

			systemAdminJwt = utils.GenerateTestJwt(fmt.Sprintf("oauth|%d", utils.GetNextUniqueId()))

			key := make([]byte, 64)
			systemAdminClaims := utils.ParseTestJwt(systemAdminJwt, key)
			baseServer.Dao.InitUserFromInviteCode(v, systemAdminClaims["sub"].(string))
		}
	}
}

func testCreateRootOrg(server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/organizations")
			jsonReq := make(map[string]interface{})
			jsonReq["Name"] = "RootOrg1024"
			req.Body = createJsonRequest(jsonReq)

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusOK got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			json.NewDecoder(resp.Body).Decode(&jsonResp)
		}

	}
}

func TestServer(t *testing.T) {
	for _, fn := range initialUserJwtFiles {
		uj, err := ioutil.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		fixedJwts = append(fixedJwts, strings.TrimSpace(string(uj)))
	}
	baseServer = server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	server := httptest.NewServer(engine)
	defer server.Close()
	t.Run("foo=bar", testBootstrap(server))
	t.Run("foo=bar", testCreateRootOrg(server))
}

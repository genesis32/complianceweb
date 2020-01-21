package integration_tests

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/genesis32/complianceweb/server"
)

type genericJson map[string]interface{}

var initialUserJwtFiles = []string{"data/105843250540508297717.txt"}
var fixedJwts []string
var systemAdminJwt string
var user0Jwt string

func testBootstrap(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		req := createBaseRequest(t, server, fixedJwts[0], "POST", "/system/bootstrap")

		jsonReq := make(map[string]interface{})
		jsonReq["SystemAdminName"] = "SystemAdmin0"
		addJsonBody(req, jsonReq)

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("creating sysadmin - statuscode expected: StatusOK got: %d", resp.StatusCode)
		}

		// Mimic the user logging in
		var jsonResp genericJson
		json.NewDecoder(resp.Body).Decode(&jsonResp)
		inviteCode := jsonResp["InviteCode"].(string)

		systemAdminJwt = simulateLogin(baseServer.Dao, inviteCode)
	}
}

func testCreateRootOrg(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		var organizationID string
		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/organizations")
			jsonReq := make(map[string]interface{})
			jsonReq["Name"] = "RootOrg1024"
			addJsonBody(req, jsonReq)

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			json.NewDecoder(resp.Body).Decode(&jsonResp)
			organizationID = jsonResp["ID"].(string)
		}

		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/users")
			jsonReq := make(map[string]interface{})
			jsonReq["Name"] = "TestUser0-" + organizationID
			jsonReq["ParentOrganizationID"] = organizationID
			jsonReq["RoleNames"] = []string{"Organization Admin"}
			addJsonBody(req, jsonReq)

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("creating user0 - statuscode expected: StatusOK got: %d", resp.StatusCode)
			}

			var jsonResp genericJson
			json.NewDecoder(resp.Body).Decode(&jsonResp)
			inviteCode := jsonResp["InviteCode"].(string)

			user0Jwt = simulateLogin(baseServer.Dao, inviteCode)
		}
	}
}

func TestBootstrapAndOrganization(t *testing.T) {
	for _, fn := range initialUserJwtFiles {
		uj, err := ioutil.ReadFile(fn)
		if err != nil {
			t.Fatal(err)
		}
		fixedJwts = append(fixedJwts, strings.TrimSpace(string(uj)))
	}
	os.Chdir("../")
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	server := httptest.NewServer(engine)
	defer server.Close()
	t.Run("testBootstrap", testBootstrap(baseServer, server))
	t.Run("testCreateRootOrg", testCreateRootOrg(baseServer, server))
}

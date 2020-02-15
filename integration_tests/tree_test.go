package integration_tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/server"
)

const (
	TreeOpAddUser   = 0
	TreeOpAddOrg    = 1
	TreeOpUserLogin = 2
	TreeOpBootstrap = 3
)

type treeOp struct {
	CallerCredentialJwt string
	Op                  int
	ParentOrgName       string
	Name                string
	Roles               []string
	HttpExpectedStatus  int
	httpResponseData    genericJson
}

func testRunner(opsToRun []treeOp, baseServer *server.Server, s *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		var credentials = map[string]string{}
		var orgNameToID = make(map[string]int64)
		var usernameToID = make(map[string]int64)
		for i := range opsToRun {
			cl := s.Client()
			switch opsToRun[i].Op {
			case TreeOpBootstrap:
				{
					cl := s.Client()
					req := createBaseRequest(t, s, "", "POST", "/system/bootstrap")
					addJsonBody(req, map[string]interface{}{
						"SystemAdminName": opsToRun[i].Name,
					})

					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("bootstrap - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {
						var jsonResp genericJson
						if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
							t.Fatal(errs)
						}
						inviteCode := jsonResp["InviteCode"].(string)
						credentials[opsToRun[i].Name] = simulateLogin(baseServer.Dao, inviteCode)
					}
				}
			case TreeOpAddOrg:
				{
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "POST", "/api/organizations")
					addJsonBody(req, map[string]interface{}{
						"Name": opsToRun[i].Name,
					})

					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("add org - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {
						var jsonResp genericJson
						if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
							t.Fatal(errs)
						}
						if v, errs := utils.StringToInt64(jsonResp["ID"].(string)); errs != nil {
							t.Fatal(err)
						} else {
							orgNameToID[opsToRun[i].Name] = v
						}
					}
				}

			case TreeOpAddUser:
				{
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "POST", "/api/users")
					addJsonBody(req, map[string]interface{}{
						"Name":                 opsToRun[i].Name,
						"ParentOrganizationID": strconv.FormatInt(orgNameToID[opsToRun[i].ParentOrgName], 10),
						"RoleNames":            opsToRun[i].Roles,
					})

					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("add user - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {
						var jsonResp genericJson
						if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
							t.Fatal(errs)
						}
						inviteCode := jsonResp["InviteCode"].(string)
						if v, errs := utils.StringToInt64(jsonResp["UserID"].(string)); errs != nil {
							t.Fatal(err)
						} else {
							usernameToID[opsToRun[i].Name] = v
							credentials[opsToRun[i].Name] = simulateLogin(baseServer.Dao, inviteCode)
						}
					}
				}
			case TreeOpUserLogin:
			}
		}
	}
}

var test0 = []treeOp{
	{
		CallerCredentialJwt: "",
		Op:                  TreeOpBootstrap,
		Name:                "SystemAdmin",
		HttpExpectedStatus:  http.StatusOK,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "",
		Name:                "RootOrg",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg",
		Name:                "RootUser",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
}

var test1 = []treeOp{
	{
		CallerCredentialJwt: "",
		Op:                  TreeOpBootstrap,
		Name:                "SystemAdmin",
		HttpExpectedStatus:  http.StatusOK,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "",
		Name:                "RootOrg",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg",
		Name:                "RootUser",
		Roles:               []string{"Org Admin"},
		HttpExpectedStatus:  http.StatusBadRequest,
	},
}

func TestTree(t *testing.T) {
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	httpServer := httptest.NewServer(engine)
	defer httpServer.Close()
	t.Run("test0 - basic tree", testRunner(test0, baseServer, httpServer))
	t.Run("test1 - status bad request", testRunner(test0, baseServer, httpServer))
}

package integration_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/server"
)

const (
	TreeOpAddUser    = 0
	TreeOpAddOrg     = 1
	TreeOpUserLogin  = 2
	TreeOpBootstrap  = 3
	TreeOpUpdateRole = 4
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

					if v, ok := orgNameToID[opsToRun[i].ParentOrgName]; ok {
						addJsonBody(req, map[string]interface{}{
							"ParentOrganizationID": fmt.Sprintf("%d", v),
							"Name":                 opsToRun[i].Name,
						})
					} else {
						addJsonBody(req, map[string]interface{}{
							"Name": opsToRun[i].Name,
						})
					}

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
			case TreeOpUpdateRole:
				{

				}
			}
		}
	}
}

var baseCreateTest = []treeOp{
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

var test2 = []treeOp{
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
		Name:                "RootOrg0",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "",
		Name:                "RootOrg1",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0Admin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0SubOrg0",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg1",
		Name:                "RootOrg1Admin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusUnauthorized,
	},
}

var test5 = []treeOp{
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
		Name:                "RootOrg0",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0Admin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0SubOrg0",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0SubOrg0",
		Name:                "RootOrg0SubOrgAdmin0",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0SubOrgAdmin0",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0-BadAdmin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusUnauthorized,
	},
}

func TestTree(t *testing.T) {
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	httpServer := httptest.NewServer(engine)
	defer httpServer.Close()
	t.Run("baseCreateTest - basic tree", testRunner(baseCreateTest, baseServer, httpServer))
	t.Run("test1 : status bad request", testRunner(test1, baseServer, httpServer))
	t.Run("test1 : unauthorized creations : lateral", testRunner(test2, baseServer, httpServer))
	t.Run("test1 : unauthorized creations : parent", testRunner(test5, baseServer, httpServer))
}

package integration_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/genesis32/complianceweb/utils"

	"github.com/genesis32/complianceweb/server"
)

type genericJson map[string]interface{}

const (
	TreeOpAddUser           = 0
	TreeOpAddOrg            = 1
	TreeOpUserLogin         = 2
	TreeOpBootstrap         = 3
	TreeOpUpdateRole        = 4
	TreeOpListOrganizations = 5
	TreeOpDeactivateUser    = 6
	TreeOpActivateUser      = 7
	TreeOpMeDetails         = 8
)

type treeOp struct {
	CallerCredentialJwt string
	Op                  int
	ParentOrgName       string
	Name                string
	Roles               []string
	SimulateLogin       bool
	HttpExpectedStatus  int
	ResponseBody        string
	ValidateFunc        func(t *testing.T, o *treeOp)
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
							if opsToRun[i].SimulateLogin {
								credentials[opsToRun[i].Name] = simulateLogin(baseServer.Dao, inviteCode)
							} else {
								credentials[opsToRun[i].Name] = generateTestJwt()
							}
						}
					}
				}
			case TreeOpUpdateRole:
				{

				}
			case TreeOpListOrganizations:
				{
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "GET", "/api/organizations")
					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("add user - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {
						if v, errs := ioutil.ReadAll(resp.Body); errs != nil {
							t.Fatal(errs)
						} else {
							opsToRun[i].ResponseBody = string(v)
							if opsToRun[i].ValidateFunc != nil {
								opsToRun[i].ValidateFunc(t, &opsToRun[i])
							}
						}
					}
				}
			case TreeOpActivateUser:
				{
					p := fmt.Sprintf("/api/users/%d", usernameToID[opsToRun[i].Name])
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "PUT", p)
					addJsonBody(req, map[string]interface{}{
						"Active": true,
					})
					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("add user - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {

					}
				}
			case TreeOpDeactivateUser:
				{
					p := fmt.Sprintf("/api/users/%d", usernameToID[opsToRun[i].Name])
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "PUT", p)
					addJsonBody(req, map[string]interface{}{
						"Active": false,
					})
					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("add user - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {

					}
				}
			case TreeOpMeDetails:
				{
					req := createBaseRequest(t, s, credentials[opsToRun[i].CallerCredentialJwt], "GET", "/api/me")
					resp, err := cl.Do(req)
					if err != nil {
						t.Fatal(err)
					}
					if resp.StatusCode != opsToRun[i].HttpExpectedStatus {
						t.Fatalf("me details - statuscode expected: %d got: %d", opsToRun[i].HttpExpectedStatus, resp.StatusCode)
					}
					if opsToRun[i].HttpExpectedStatus >= 200 && opsToRun[i].HttpExpectedStatus < 300 {

					}
				}
			}
		}
	}
}

var baseTree = []treeOp{
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
		SimulateLogin:       true,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0Admin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
}

var invalidRoleTest = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		SimulateLogin:       false,
		Name:                "RootOrg0Admin1",
		Roles:               []string{"Org Admin"},
		HttpExpectedStatus:  http.StatusBadRequest,
	},
}...)

var unauthorizedLateralRole = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "SystemAdmin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "",
		Name:                "RootOrg1",
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
		SimulateLogin:       true,
		Name:                "RootOrg1Admin",
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusUnauthorized,
	},
}...)

var unauthorizedParentTest = append(baseTree, []treeOp{
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
		SimulateLogin:       true,
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0SubOrgAdmin0",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0-BadAdmin",
		SimulateLogin:       true,
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusUnauthorized,
	},
}...)

var listOrganizationsTest = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0SubOrg0",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0SubOrg1",
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpListOrganizations,
		HttpExpectedStatus:  http.StatusOK,
		ValidateFunc: func(t *testing.T, o *treeOp) {
			var jsonResp genericJson
			buff := bytes.NewBufferString(o.ResponseBody)
			if errs := json.NewDecoder(buff).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			if len(jsonResp["Children"].([]interface{})) != 2 {
				t.Fatal("")
			}
		},
	},
}...)

var invalidLoginTest = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0User1",
		SimulateLogin:       false,
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0User1",
		Op:                  TreeOpAddOrg,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0SubOrg1",
		HttpExpectedStatus:  http.StatusForbidden,
	},
}...)

var meDetailsUserTest = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0User1",
		SimulateLogin:       true,
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0User1",
		Op:                  TreeOpMeDetails,
		HttpExpectedStatus:  http.StatusOK,
	},
}...)

var deactivateUserTest = append(baseTree, []treeOp{
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpAddUser,
		ParentOrgName:       "RootOrg0",
		Name:                "RootOrg0User1",
		SimulateLogin:       true,
		Roles:               []string{"Organization Admin"},
		HttpExpectedStatus:  http.StatusCreated,
	},
	{
		CallerCredentialJwt: "RootOrg0User1",
		Op:                  TreeOpListOrganizations,
		HttpExpectedStatus:  http.StatusOK,
	},
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpDeactivateUser,
		Name:                "RootOrg0User1",
		HttpExpectedStatus:  http.StatusOK,
	},
	{
		CallerCredentialJwt: "RootOrg0User1",
		Op:                  TreeOpListOrganizations,
		HttpExpectedStatus:  http.StatusForbidden,
	},
	{
		CallerCredentialJwt: "RootOrg0SubOrgAdmin0",
		Op:                  TreeOpDeactivateUser,
		Name:                "RootOrg0Admin",
		HttpExpectedStatus:  http.StatusUnauthorized,
	},
}...)

var activateUserTest = append(deactivateUserTest, []treeOp{
	{
		CallerCredentialJwt: "RootOrg0Admin",
		Op:                  TreeOpActivateUser,
		Name:                "RootOrg0User1",
		HttpExpectedStatus:  http.StatusOK,
	},
	{
		CallerCredentialJwt: "RootOrg0User1",
		Op:                  TreeOpListOrganizations,
		HttpExpectedStatus:  http.StatusOK,
	},
}...)

func TestTree(t *testing.T) {
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	httpServer := httptest.NewServer(engine)
	defer httpServer.Close()
	t.Run("baseTree - basic tree", testRunner(baseTree, baseServer, httpServer))
	t.Run("invalidRoleTest : status bad request", testRunner(invalidRoleTest, baseServer, httpServer))
	t.Run("invalidRoleTest : unauthorized creations : lateral", testRunner(unauthorizedLateralRole, baseServer, httpServer))
	t.Run("invalidRoleTest : unauthorized creations : parent", testRunner(unauthorizedParentTest, baseServer, httpServer))
	t.Run("list organizations", testRunner(listOrganizationsTest, baseServer, httpServer))
	t.Run("invalid login", testRunner(invalidLoginTest, baseServer, httpServer))
	t.Run("deactivate user", testRunner(deactivateUserTest, baseServer, httpServer))
	t.Run("activate user", testRunner(activateUserTest, baseServer, httpServer))
	t.Run("user details", testRunner(meDetailsUserTest, baseServer, httpServer))
}

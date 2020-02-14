package integration_tests

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/genesis32/complianceweb/server"
)

type genericJson map[string]interface{}

var (
	initialUserJwtFiles                  = []string{"data/105843250540508297717.txt"}
	fixedJwts                            []string
	systemAdminJwt                       string
	orgAdminJwt                          string
	subOrgAdminJwt                       string
	gcpAdminUser0Jwt                     string
	rootOrganization0                    string
	rootOrganization1                    string
	subOrganization0ForRootOrganization0 string
)

func testBootstrap(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		req := createBaseRequest(t, server, fixedJwts[0], "POST", "/system/bootstrap")
		addJsonBody(req, map[string]interface{}{
			"SystemAdminName": "SystemAdmin0",
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("creating sysadmin - statuscode expected: StatusOK got: %d", resp.StatusCode)
		}

		// Mimic the user logging in
		var jsonResp genericJson
		if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
			t.Fatal(errs)
		}
		inviteCode := jsonResp["InviteCode"].(string)

		systemAdminJwt = simulateLogin(baseServer.Dao, inviteCode)
	}
}

func testCreateRootOrg(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		// Create a base organization
		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/organizations")
			addJsonBody(req, map[string]interface{}{
				"Name": "RootOrg1024",
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			rootOrganization0 = jsonResp["ID"].(string)
		}

		// create a second organization tree
		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/organizations")
			addJsonBody(req, map[string]interface{}{
				"Name": "RootOrg2048",
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			rootOrganization1 = jsonResp["ID"].(string)
		}

		// Create the first admin user of Organization1024 - user0
		{
			req := createBaseRequest(t, server, systemAdminJwt, "POST", "/api/users")
			addJsonBody(req, map[string]interface{}{
				"Name":                 "OrgAdmin0-" + rootOrganization0,
				"ParentOrganizationID": rootOrganization0,
				"RoleNames":            []string{"Organization Admin"},
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}

			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("creating user0 - statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}

			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			inviteCode := jsonResp["InviteCode"].(string)

			orgAdminJwt = simulateLogin(baseServer.Dao, inviteCode)
		}

		// user0 create an organization in the other tree (should fail)
		{
			req := createBaseRequest(t, server, orgAdminJwt, "POST", "/api/organizations")
			addJsonBody(req, map[string]interface{}{
				"ParentOrganizationID": rootOrganization1,
				"Name":                 "RootOrg2048-0",
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("statuscode expected: StatusUnauthorized got: %d", resp.StatusCode)
			}
		}

		// create an organization under the one the user is an admin for.
		{
			req := createBaseRequest(t, server, orgAdminJwt, "POST", "/api/organizations")
			addJsonBody(req, map[string]interface{}{
				"ParentOrganizationID": rootOrganization0,
				"Name":                 "RootOrg1024-0",
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			subOrganization0ForRootOrganization0 = jsonResp["ID"].(string)
		}

		// create a user with an organization admin role
		{
			req := createBaseRequest(t, server, orgAdminJwt, "POST", "/api/users")
			addJsonBody(req, map[string]interface{}{
				"Name":                 "GCPAdminUser0-" + subOrganization0ForRootOrganization0,
				"ParentOrganizationID": subOrganization0ForRootOrganization0,
				"RoleNames":            []string{"Organization Admin"},
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			inviteCode := jsonResp["InviteCode"].(string)

			subOrgAdminJwt = simulateLogin(baseServer.Dao, inviteCode)
		}

		// create a user with just a gcp admin role
		{
			req := createBaseRequest(t, server, orgAdminJwt, "POST", "/api/users")
			addJsonBody(req, map[string]interface{}{
				"Name":                 "GCPAdminUser0-" + subOrganization0ForRootOrganization0,
				"ParentOrganizationID": subOrganization0ForRootOrganization0,
				"RoleNames":            []string{"GCP Administrator"},
			})

			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != http.StatusCreated {
				t.Fatalf("statuscode expected: StatusCreated got: %d", resp.StatusCode)
			}
			var jsonResp genericJson
			if errs := json.NewDecoder(resp.Body).Decode(&jsonResp); errs != nil {
				t.Fatal(errs)
			}
			inviteCode := jsonResp["InviteCode"].(string)

			gcpAdminUser0Jwt = simulateLogin(baseServer.Dao, inviteCode)
		}

	}
}

// user with just a gcp role is now trying to create a user in their org (it should fail)
func testNoUserCreateRole(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		req := createBaseRequest(t, server, gcpAdminUser0Jwt, "POST", "/api/users")
		addJsonBody(req, map[string]interface{}{
			"Name":                 "TestUser1-" + rootOrganization0,
			"ParentOrganizationID": rootOrganization0,
			"RoleNames":            []string{"GCP Administrator"},
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			bodyBytes, _ := ioutil.ReadAll(resp.Body)
			t.Fatalf("statuscode expected: Unauthorized got: %d body: %v", resp.StatusCode, string(bodyBytes))
		}
	}
}

// invalid role name should fail
func testCreateInvalidRole(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()

		// Create a base organization
		req := createBaseRequest(t, server, orgAdminJwt, "POST", "/api/users")
		addJsonBody(req, map[string]interface{}{
			"Name":                 "TestUser1-" + rootOrganization0,
			"ParentOrganizationID": rootOrganization0,
			"RoleNames":            []string{"GCP Admin"},
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("statuscode expected: StatusBadRequest got: %d", resp.StatusCode)
		}
	}
}

// sub org admin tries to create a user in the org above where they are admin
func testSubOrgAdminCreateUserInParent(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()

		// Create a base organization
		req := createBaseRequest(t, server, subOrgAdminJwt, "POST", "/api/users")
		addJsonBody(req, map[string]interface{}{
			"Name":                 "TestUser1-" + rootOrganization0,
			"ParentOrganizationID": rootOrganization0,
			"RoleNames":            []string{"GCP Administrator"},
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("statuscode expected: StatusUnauthorized got: %d", resp.StatusCode)
		}
	}
}

// sub org admin tries to create a user in the org above where they are admin
func testSubOrgAdminCreateOrgInParent(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()

		// Create a base organization
		req := createBaseRequest(t, server, subOrgAdminJwt, "POST", "/api/organizations")
		addJsonBody(req, map[string]interface{}{
			"Name":                 "SubOrg1024-" + rootOrganization0,
			"ParentOrganizationID": rootOrganization0,
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("statuscode expected: StatusUnauthorized got: %d", resp.StatusCode)
		}
	}
}

// cannot add a gcp service account if we don't have the required metadata
func testCreateGcpServiceAccountNoMetadata(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		// Create a base organization
		path := fmt.Sprintf("/api/resources/%s/gcp.serviceaccount", subOrganization0ForRootOrganization0)
		req := createBaseRequest(t, server, gcpAdminUser0Jwt, "POST", path)
		addJsonBody(req, map[string]interface{}{
			"Name": "serviceAccount-" + subOrganization0ForRootOrganization0,
		})
		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatalf("statuscode expected: StatusBadRequest got: %d", resp.StatusCode)
		}
	}
}

// org admins can't create gcp accounts
func testNotAuthorizedCreateGcpServiceAccount(baseServer *server.Server, server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		// Create a base organization
		path := fmt.Sprintf("/api/resources/%s/gcp.serviceaccount", subOrganization0ForRootOrganization0)
		req := createBaseRequest(t, server, orgAdminJwt, "POST", path)
		addJsonBody(req, map[string]interface{}{
			"Name": "serviceAccount0-" + subOrganization0ForRootOrganization0,
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("statuscode expected: StatusUnauthorized got: %d", resp.StatusCode)
		}
	}
}

func updateMetadata(t *testing.T, cl *http.Client, s *httptest.Server, org string, data *server.OrganizationMetadataUpdateRequest) {
	path := fmt.Sprintf("/api/organizations/%s/metadata", org)
	// TODO: Move orgAdminJwt out of this.
	req := createBaseRequest(t, s, orgAdminJwt, "PUT", path)
	addJsonBody(req, data)

	resp, err := cl.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("statuscode expected: StatusOK got: %d", resp.StatusCode)
	}
}

// set and get metadata a System Administrator
func testUpdateOrganizationMetadata(baseServer *server.Server, s *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := s.Client()

		data := &server.OrganizationMetadataUpdateRequest{Metadata: map[string]interface{}{"foo": "bar"}}
		updateMetadata(t, cl, s, subOrganization0ForRootOrganization0, data)

		path := fmt.Sprintf("/api/organizations/%s/metadata", subOrganization0ForRootOrganization0)
		req := createBaseRequest(t, s, gcpAdminUser0Jwt, "GET", path)

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("statuscode expected: StatusOK got: %d", resp.StatusCode)
		}
		var response server.OrganizationMetadataResponse
		if errs := json.NewDecoder(resp.Body).Decode(&response); errs != nil {
			t.Fatal(errs)
		}
		metadataValue := response.Metadata["foo"].(string)
		if metadataValue != "bar" {
			t.Fatalf("metadata: %v does not match", metadataValue)
		}
	}
}

/*
TODO Tests: verify metadata operations, deactivate user, verify audit records produced, maybe all metadata
permissions in second tree so we can then verify vs. first tree, lack visibility of users under org tree,
verify tree listings (Programatically create a tree to run tests under after these basic sanity checks)

gcp admin can create a service account in the orgs they are in or under

TODO: Have to mock gcp endpoints or something if we want to tes this, but might just be a better idea to create
register a test resource with some permissions and use that for this level of testing and put integration tests
with service providers in another tag.
*/

func testCreateGcpServiceAccount(baseServer *server.Server, s *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := s.Client()

		data := &server.OrganizationMetadataUpdateRequest{Metadata: map[string]interface{}{"gcpCredentials": "{}"}}
		updateMetadata(t, cl, s, subOrganization0ForRootOrganization0, data)

		// Create a base organization
		path := fmt.Sprintf("/api/resources/%s/gcp.serviceaccount", subOrganization0ForRootOrganization0)
		req := createBaseRequest(t, s, gcpAdminUser0Jwt, "POST", path)
		addJsonBody(req, map[string]interface{}{
			"Name": "serviceAccount-" + subOrganization0ForRootOrganization0,
		})

		resp, err := cl.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("statuscode expected: StatusOK got: %d", resp.StatusCode)
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
	if errs := os.Chdir("../"); errs != nil {
		t.Fatal(errs)
	}
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()

	httpServer := httptest.NewServer(engine)
	defer httpServer.Close()
	t.Run("testBootstrap", testBootstrap(baseServer, httpServer))
	t.Run("testCreateRootOrg", testCreateRootOrg(baseServer, httpServer))
	t.Run("testNoUserCreateRole", testNoUserCreateRole(baseServer, httpServer))
	t.Run("testCreateInvalidRole", testCreateInvalidRole(baseServer, httpServer))
	t.Run("testSubOrgAdminCreateUserInParent", testSubOrgAdminCreateUserInParent(baseServer, httpServer))
	t.Run("testSubOrgAdminCreateOrgInParent", testSubOrgAdminCreateOrgInParent(baseServer, httpServer))
	t.Run("not authorized to create gcp service account", testNotAuthorizedCreateGcpServiceAccount(baseServer, httpServer))
	t.Run("update organization metadata", testUpdateOrganizationMetadata(baseServer, httpServer))

	t.Run("org admin cannot create gcp service account", testNotAuthorizedCreateGcpServiceAccount(baseServer, httpServer))
	t.Run("gcp service account no metadata", testCreateGcpServiceAccountNoMetadata(baseServer, httpServer))
	//	t.Run("gcp service account w/ metadata", testCreateGcpServiceAccount(baseServer, httpServer))
}

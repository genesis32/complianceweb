package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/genesis32/complianceweb/server"
)

type genericJson map[string]interface{}

var bearerJwt0 string

func testStuff(server *httptest.Server) func(t *testing.T) {
	return func(t *testing.T) {
		cl := server.Client()
		{
			req, err := http.NewRequest("GET", server.URL+"/api/organizations", nil)
			if err != nil {
				t.Fatal(err)
			}
			fmt.Print(string(bearerJwt0))
			req.Header.Add("Authorization", "Bearer "+string(bearerJwt0))
			resp, err := cl.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			if resp.StatusCode != 200 {
				t.Fatal("status code exported 200 got #{resp.StatusCode}")
			}
			var jsonResp genericJson
			json.NewDecoder(resp.Body).Decode(&jsonResp)
			fmt.Printf("%+v", jsonResp)
		}

	}
}

func TestServer(t *testing.T) {
	var err error
	b, err := ioutil.ReadFile("test/data/105843250540508297717.txt")
	if err != nil {
		t.Fatal(err)
	}
	bearerJwt0 = strings.TrimSpace(string(b))
	baseServer := server.NewServer()
	defer baseServer.Shutdown()
	engine := baseServer.Initialize()
	server := httptest.NewServer(engine)
	defer server.Close()
	t.Run("foo=bar", testStuff(server))
}

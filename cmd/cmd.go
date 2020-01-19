package cmd

import (
	"fmt"
	"log"

	"github.com/dgrijalva/jwt-go"

	"github.com/genesis32/complianceweb/server"
	"github.com/spf13/cobra"
)

func init() {
	RootCmd.AddCommand(GenerateJwtCommand)

	GenerateJwtCommand.Flags().StringP("sub", "s", "provider | subjectid", "subject field")
}

var GenerateJwtCommand = &cobra.Command{
	Use: "generatejwt",
	Run: func(cmd *cobra.Command, args []string) {

		sub, err := cmd.Flags().GetString("sub")
		if err != nil {
			panic(err)
		}

		fmt.Println("sub:", sub)
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
		foobar, err := token.SignedString(key)
		if err != nil {
			panic(err)
		}

		fmt.Println(foobar)
	},
}

var RootCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		server := server.NewServer()
		err := server.Startup()
		if err != nil {
			log.Fatal(err)
		}
		defer server.Shutdown()

		server.Serve()
	},
}

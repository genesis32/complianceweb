package cmd

import (
	"fmt"

	"github.com/genesis32/complianceweb/utils"

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

		ret := utils.GenerateTestJwt(sub)

		fmt.Println(ret)
	},
}

var RootCmd = &cobra.Command{
	Use: "server",
	Run: func(cmd *cobra.Command, args []string) {
		server := server.NewServer()
		defer server.Shutdown()

		server.Initialize()
		server.Serve()
	},
}

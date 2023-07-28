package main

import (
	"os"

	_ "github.com/KimMachineGun/automemlimit"
	"github.com/cosmos/cosmos-sdk/server"
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"
	_ "go.uber.org/automaxprocs"

	"github.com/NibiruChain/nibiru/app"
	"github.com/NibiruChain/nibiru/cmd/nibid/cmd"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

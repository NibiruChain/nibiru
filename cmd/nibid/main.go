package main

import (
	svrcmd "github.com/cosmos/cosmos-sdk/server/cmd"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/cmd/nibid/cmd"
)

func main() {
	rootCmd, _ := cmd.NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		panic(err)
	}
}

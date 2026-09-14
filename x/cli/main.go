package main

import (
	"os"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server"
	svrcmd "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/server/cmd"

	"github.com/NibiruChain/nibiru/v2/app"
)

func main() {
	rootCmd, _ := NewRootCmd()

	if err := svrcmd.Execute(rootCmd, "", app.DefaultNodeHome); err != nil {
		switch e := err.(type) {
		case server.ErrorCode:
			os.Exit(e.Code)

		default:
			os.Exit(1)
		}
	}
}

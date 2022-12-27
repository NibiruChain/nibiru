package cligen

import (
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
	"reflect"
	"strings"
)

type CliGen struct {
	m      sdk.Msg
	params Params
}

func NewCliGen() CliGen {
	return CliGen{}
}

func (c CliGen) ForMessage(msg sdk.Msg) CliGen {
	c.m = msg

	return c
}

func (c CliGen) WithParams(params Params) CliGen {
	c.params = params

	return c
}

func (c CliGen) Generate() *cobra.Command {
	cmd := &cobra.Command{}

	mandatoryParams := c.params.Mandatory()
	cmd.Args = cobra.ExactArgs(len(mandatoryParams))

	cmd.Use = c.genUse()

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		clientCtx, err := client.GetClientTxContext(cmd)
		if err != nil {
			return err
		}

		msg := c.buildMsg()

		return tx.GenerateOrBroadcastTxCLI(clientCtx, cmd.Flags(), msg)
	}

	flags.AddTxFlagsToCmd(cmd)

	return cmd
}

func (c CliGen) buildMsg() sdk.Msg {
	return c.m
}

func (c CliGen) genUse() string {

	nameMsg := reflect.ValueOf(c.m).Elem().Type().Name()
	nameWithoutPrefixAndLowCase := strings.ToLower(strings.TrimPrefix(nameMsg, "Msg"))

	var b strings.Builder

	b.WriteString(nameWithoutPrefixAndLowCase)
	b.WriteString(" ")

	mandatoryParams := c.params.Mandatory()
	for i, param := range mandatoryParams {
		b.WriteString("[")
		b.WriteString(param.Name)
		b.WriteString("]")

		if i < len(mandatoryParams)-1 {
			b.WriteString(" ")
		}
	}

	return b.String()
}

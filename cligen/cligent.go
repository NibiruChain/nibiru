package cligen

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/cobra"
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

func (c CliGen) Generate() cobra.Command {
	return cobra.Command{}
}

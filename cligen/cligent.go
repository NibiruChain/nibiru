package cligen

import sdk "github.com/cosmos/cosmos-sdk/types"

type Param struct {
	Name string
	Type string
}

type Params []Param

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

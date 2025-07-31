package cli_test

import (
	"fmt"
	"math/big"

	gethcommon "github.com/ethereum/go-ethereum/common"

	"github.com/NibiruChain/nibiru/v2/x/evm"
	"github.com/NibiruChain/nibiru/v2/x/evm/evmtest"
)

var (
	dummyAccs     = evmtest.NewEthPrivAccs(3)
	dummyEthAddr  = dummyAccs[1].EthAddr.Hex()
	dummyFuntoken = evm.NewFunToken(
		gethcommon.BigToAddress(big.NewInt(123)),
		"ibc/testtoken",
		false,
	)
)

func (s *Suite) TestCmdConvertCoinToEvm() {
	testCases := []TestCase{
		{
			name: "happy: convert-coin-to-evm",
			args: []string{
				"convert-coin-to-evm",
				dummyEthAddr,
				fmt.Sprintf("%d%s", 123, dummyFuntoken.BankDenom),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: coin format",
			args: []string{
				"convert-coin-to-evm",
				dummyAccs[1].EthAddr.Hex(),
				fmt.Sprintf("%s %d", dummyFuntoken.BankDenom, 123),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "invalid decimal coin expression",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s)
	}
}

func (s *Suite) TestCmdConvertEvmToCoin() {
	testCases := []TestCase{
		{
			name: "happy: convert-evm-to-coin",
			args: []string{
				"convert-evm-to-coin",
				dummyFuntoken.Erc20Addr.String(),
				"123456",
				s.testAcc.Address.String(),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: invalid erc20 address",
			args: []string{
				"convert-evm-to-coin",
				"not-an-address",
				"123456",
				s.testAcc.Address.String(),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "invalid ERC20 contract address",
		},
		{
			name: "sad: invalid amount",
			args: []string{
				"convert-evm-to-coin",
				dummyFuntoken.Erc20Addr.String(),
				"not-a-number",
				s.testAcc.Address.String(),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "invalid amount",
		},
		{
			name: "sad: invalid recipient address",
			args: []string{
				"convert-evm-to-coin",
				dummyFuntoken.Erc20Addr.String(),
				"123456",
				"invalid-address",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "invalid recipient address",
		},
		{
			name: "sad: missing args",
			args: []string{
				"convert-evm-to-coin",
				dummyFuntoken.Erc20Addr.String(),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "accepts 3 arg(s), received 1",
		},
		{
			name: "sad: too many args",
			args: []string{
				"convert-evm-to-coin",
				dummyFuntoken.Erc20Addr.String(),
				"123456",
				s.testAcc.Address.String(),
				"extra-arg",
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "accepts 3 arg(s), received 4",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s)
	}
}

func (s *Suite) TestCmdCreateFunToken() {
	testCases := []TestCase{
		{
			name: "happy: create-funtoken (erc20)",
			args: []string{
				"create-funtoken",
				fmt.Sprintf("--erc20=%s", dummyEthAddr),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "happy: create-funtoken (bank coin)",
			args: []string{
				"create-funtoken",
				fmt.Sprintf("--bank-denom=%s", dummyFuntoken.BankDenom),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "",
		},
		{
			name: "sad: too many args",
			args: []string{
				"create-funtoken",
				fmt.Sprintf("--erc20=%s", dummyEthAddr),
				fmt.Sprintf("--bank-denom=%s", dummyFuntoken.BankDenom),
			},
			extraArgs: []string{fmt.Sprintf("--from=%s", s.testAcc.Address)},
			wantErr:   "exactly one of the flags --bank-denom or --erc20 must be specified",
		},
	}

	for _, tc := range testCases {
		tc.RunTxCmd(s)
	}
}

func (s *Suite) TestCmdQueryAccount() {
	testCases := []TestCase{
		{
			name: "happy: query account (bech32)",
			args: []string{
				"account",
				dummyAccs[0].NibiruAddr.String(),
			},
			wantErr: "",
		},
		{
			name: "happy: query account (eth hex)",
			args: []string{
				"account",
				dummyAccs[0].EthAddr.Hex(),
			},
			wantErr: "",
		},
		{
			name: "happy: query account (eth hex) --offline",
			args: []string{
				"account",
				dummyAccs[0].EthAddr.Hex(),
				"--offline",
			},
			wantErr: "",
		},
		{
			name: "happy: query account (bech32) --offline",
			args: []string{
				"account",
				dummyAccs[0].NibiruAddr.String(),
				"--offline",
			},
			wantErr: "",
		},
		{
			name: "sad: too many args",
			args: []string{
				"funtoken",
				"arg1",
				"arg2",
			},
			wantErr: "accepts 1 arg",
		},
	}

	for _, tc := range testCases {
		tc.RunQueryCmd(s)
	}
}

func (s *Suite) TestCmdQueryFunToken() {
	testCases := []TestCase{
		{
			name: "happy: query funtoken (bank coin denom)",
			args: []string{
				"funtoken",
				dummyFuntoken.BankDenom,
			},
			wantErr: "",
		},
		{
			name: "happy: query funtoken (erc20 addr)",
			args: []string{
				"funtoken",
				dummyFuntoken.Erc20Addr.String(),
			},
			wantErr: "",
		},
		{
			name: "sad: too many args",
			args: []string{
				"funtoken",
				"arg1",
				"arg2",
			},
			wantErr: "accepts 1 arg",
		},
	}

	for _, tc := range testCases {
		tc.RunQueryCmd(s)
	}
}

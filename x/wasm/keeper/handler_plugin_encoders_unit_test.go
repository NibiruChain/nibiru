package keeper

import (
	"testing"

	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	ibctransfertypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/apps/transfer/types"
	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	channeltypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/04-channel/types"

	codectypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec/types"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/distribution/types"
	v1 "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types/v1"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"

	"github.com/NibiruChain/nibiru/v2/x/wasm/keeper/wasmtesting"
	"github.com/NibiruChain/nibiru/v2/x/wasm/types"
)

func TestEncoding(t *testing.T) {
	var (
		addr1       = RandomAccountAddress(t)
		addr2       = RandomAccountAddress(t)
		addr3       = RandomAccountAddress(t)
		invalidAddr = "xrnd1d02kd90n38qvr3qb9qof83fn2d2"
	)
	valAddr := make(sdk.ValAddress, types.SDKAddrLen)
	valAddr[0] = 12
	valAddr2 := make(sdk.ValAddress, types.SDKAddrLen)
	valAddr2[1] = 123

	jsonMsg := types.RawContractMessage(`{"foo": 123}`)

	bankMsg := &banktypes.MsgSend{
		FromAddress: addr2.String(),
		ToAddress:   addr1.String(),
		Amount: sdk.Coins{
			sdk.NewInt64Coin("uatom", 12345),
			sdk.NewInt64Coin("utgd", 54321),
		},
	}
	bankMsgBin, err := proto.Marshal(bankMsg)
	require.NoError(t, err)

	msg, err := codectypes.NewAnyWithValue(types.MsgStoreCodeFixture())
	require.NoError(t, err)
	proposalMsg := &v1.MsgSubmitProposal{
		Proposer:       addr1.String(),
		Messages:       []*codectypes.Any{msg},
		InitialDeposit: sdk.NewCoins(sdk.NewInt64Coin("uatom", 12345)),
		Title:          "proposal",
		Summary:        "proposal summary",
	}
	proposalMsgBin, err := proto.Marshal(proposalMsg)
	require.NoError(t, err)

	cases := map[string]struct {
		sender             sdk.AccAddress
		srcMsg             wvm.CosmosMsg
		srcContractIBCPort string
		transferPortSource types.ICS20TransferPortSource
		// set if valid
		output []sdk.Msg
		// set if expect mapping fails
		expError bool
		// set if sdk validate basic should fail
		expInvalid bool
	}{
		"simple send": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Bank: &wvm.BankMsg{
					Send: &wvm.SendMsg{
						ToAddress: addr2.String(),
						Amount: []wvm.Coin{
							{
								Denom:  "uatom",
								Amount: "12345",
							},
							{
								Denom:  "usdt",
								Amount: "54321",
							},
						},
					},
				},
			},
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   addr2.String(),
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 12345),
						sdk.NewInt64Coin("usdt", 54321),
					},
				},
			},
		},
		"invalid send amount": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Bank: &wvm.BankMsg{
					Send: &wvm.SendMsg{
						ToAddress: addr2.String(),
						Amount: []wvm.Coin{
							{
								Denom:  "uatom",
								Amount: "123.456",
							},
						},
					},
				},
			},
			expError: true,
		},
		"invalid address": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Bank: &wvm.BankMsg{
					Send: &wvm.SendMsg{
						ToAddress: invalidAddr,
						Amount: []wvm.Coin{
							{
								Denom:  "uatom",
								Amount: "7890",
							},
						},
					},
				},
			},
			expError:   false, // addresses are checked in the handler
			expInvalid: true,
			output: []sdk.Msg{
				&banktypes.MsgSend{
					FromAddress: addr1.String(),
					ToAddress:   invalidAddr,
					Amount: sdk.Coins{
						sdk.NewInt64Coin("uatom", 7890),
					},
				},
			},
		},
		"wasm execute": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					Execute: &wvm.ExecuteMsg{
						ContractAddr: addr2.String(),
						Msg:          jsonMsg,
						Funds: []wvm.Coin{
							wvm.NewCoin(12, "eth"),
						},
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgExecuteContract{
					Sender:   addr1.String(),
					Contract: addr2.String(),
					Msg:      jsonMsg,
					Funds:    sdk.NewCoins(sdk.NewInt64Coin("eth", 12)),
				},
			},
		},
		"wasm instantiate": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					Instantiate: &wvm.InstantiateMsg{
						CodeID: 7,
						Msg:    jsonMsg,
						Funds: []wvm.Coin{
							wvm.NewCoin(123, "eth"),
						},
						Label: "myLabel",
						Admin: addr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgInstantiateContract{
					Sender: addr1.String(),
					CodeID: 7,
					Label:  "myLabel",
					Msg:    jsonMsg,
					Funds:  sdk.NewCoins(sdk.NewInt64Coin("eth", 123)),
					Admin:  addr2.String(),
				},
			},
		},
		"wasm instantiate2": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					Instantiate2: &wvm.Instantiate2Msg{
						CodeID: 7,
						Msg:    jsonMsg,
						Funds: []wvm.Coin{
							wvm.NewCoin(123, "eth"),
						},
						Label: "myLabel",
						Admin: addr2.String(),
						Salt:  []byte("mySalt"),
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgInstantiateContract2{
					Sender: addr1.String(),
					Admin:  addr2.String(),
					CodeID: 7,
					Label:  "myLabel",
					Msg:    jsonMsg,
					Funds:  sdk.NewCoins(sdk.NewInt64Coin("eth", 123)),
					Salt:   []byte("mySalt"),
					FixMsg: false,
				},
			},
		},
		"wasm migrate": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					Migrate: &wvm.MigrateMsg{
						ContractAddr: addr1.String(),
						NewCodeID:    12,
						Msg:          jsonMsg,
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgMigrateContract{
					Sender:   addr2.String(),
					Contract: addr1.String(),
					CodeID:   12,
					Msg:      jsonMsg,
				},
			},
		},
		"wasm update admin": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					UpdateAdmin: &wvm.UpdateAdminMsg{
						ContractAddr: addr1.String(),
						Admin:        addr3.String(),
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgUpdateAdmin{
					Sender:   addr2.String(),
					Contract: addr1.String(),
					NewAdmin: addr3.String(),
				},
			},
		},
		"wasm clear admin": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Wasm: &wvm.WasmMsg{
					ClearAdmin: &wvm.ClearAdminMsg{
						ContractAddr: addr1.String(),
					},
				},
			},
			output: []sdk.Msg{
				&types.MsgClearAdmin{
					Sender:   addr2.String(),
					Contract: addr1.String(),
				},
			},
		},
		"staking delegate": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Staking: &wvm.StakingMsg{
					Delegate: &wvm.DelegateMsg{
						Validator: valAddr.String(),
						Amount:    wvm.NewCoin(777, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking delegate to non-validator - invalid": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Staking: &wvm.StakingMsg{
					Delegate: &wvm.DelegateMsg{
						Validator: addr2.String(),
						Amount:    wvm.NewCoin(777, "stake"),
					},
				},
			},
			expError:   false, // fails in the handler
			expInvalid: true,
			output: []sdk.Msg{
				&stakingtypes.MsgDelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: addr2.String(),
					Amount:           sdk.NewInt64Coin("stake", 777),
				},
			},
		},
		"staking undelegate": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Staking: &wvm.StakingMsg{
					Undelegate: &wvm.UndelegateMsg{
						Validator: valAddr.String(),
						Amount:    wvm.NewCoin(555, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgUndelegate{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr.String(),
					Amount:           sdk.NewInt64Coin("stake", 555),
				},
			},
		},
		"staking redelegate": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Staking: &wvm.StakingMsg{
					Redelegate: &wvm.RedelegateMsg{
						SrcValidator: valAddr.String(),
						DstValidator: valAddr2.String(),
						Amount:       wvm.NewCoin(222, "stake"),
					},
				},
			},
			output: []sdk.Msg{
				&stakingtypes.MsgBeginRedelegate{
					DelegatorAddress:    addr1.String(),
					ValidatorSrcAddress: valAddr.String(),
					ValidatorDstAddress: valAddr2.String(),
					Amount:              sdk.NewInt64Coin("stake", 222),
				},
			},
		},
		"staking withdraw (explicit recipient)": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Distribution: &wvm.DistributionMsg{
					WithdrawDelegatorReward: &wvm.WithdrawDelegatorRewardMsg{
						Validator: valAddr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgWithdrawDelegatorReward{
					DelegatorAddress: addr1.String(),
					ValidatorAddress: valAddr2.String(),
				},
			},
		},
		"staking set withdraw address": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Distribution: &wvm.DistributionMsg{
					SetWithdrawAddress: &wvm.SetWithdrawAddressMsg{
						Address: addr2.String(),
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgSetWithdrawAddress{
					DelegatorAddress: addr1.String(),
					WithdrawAddress:  addr2.String(),
				},
			},
		},
		"distribution fund community pool": {
			sender: addr1,
			srcMsg: wvm.CosmosMsg{
				Distribution: &wvm.DistributionMsg{
					FundCommunityPool: &wvm.FundCommunityPoolMsg{
						Amount: wvm.Coins{
							wvm.NewCoin(200, "stones"),
							wvm.NewCoin(200, "feathers"),
						},
					},
				},
			},
			output: []sdk.Msg{
				&distributiontypes.MsgFundCommunityPool{
					Depositor: addr1.String(),
					Amount: sdk.NewCoins(
						sdk.NewInt64Coin("stones", 200),
						sdk.NewInt64Coin("feathers", 200),
					),
				},
			},
		},
		"stargate encoded bank msg": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Stargate: &wvm.StargateMsg{
					TypeURL: "/cosmos.bank.v1beta1.MsgSend",
					Value:   bankMsgBin,
				},
			},
			output: []sdk.Msg{bankMsg},
		},
		"stargate encoded msg with any type": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Stargate: &wvm.StargateMsg{
					TypeURL: "/cosmos.gov.v1.MsgSubmitProposal",
					Value:   proposalMsgBin,
				},
			},
			output: []sdk.Msg{proposalMsg},
		},
		"stargate encoded invalid typeUrl": {
			sender: addr2,
			srcMsg: wvm.CosmosMsg{
				Stargate: &wvm.StargateMsg{
					TypeURL: "/cosmos.bank.v2.MsgSend",
					Value:   bankMsgBin,
				},
			},
			expError: true,
		},
		"IBC transfer with block timeout": {
			sender:             addr1,
			srcContractIBCPort: "myIBCPort",
			srcMsg: wvm.CosmosMsg{
				IBC: &wvm.IBCMsg{
					Transfer: &wvm.TransferMsg{
						ChannelID: "myChanID",
						ToAddress: addr2.String(),
						Amount: wvm.Coin{
							Denom:  "ALX",
							Amount: "1",
						},
						Timeout: wvm.IBCTimeout{
							Block: &wvm.IBCTimeoutBlock{Revision: 1, Height: 2},
						},
					},
				},
			},
			transferPortSource: wasmtesting.MockIBCTransferKeeper{GetPortFn: func(ctx sdk.Context) string {
				return "myTransferPort"
			}},
			output: []sdk.Msg{
				&ibctransfertypes.MsgTransfer{
					SourcePort:    "myTransferPort",
					SourceChannel: "myChanID",
					Token: sdk.Coin{
						Denom:  "ALX",
						Amount: sdk.NewInt(1),
					},
					Sender:        addr1.String(),
					Receiver:      addr2.String(),
					TimeoutHeight: clienttypes.Height{RevisionNumber: 1, RevisionHeight: 2},
				},
			},
		},
		"IBC transfer with time timeout": {
			sender:             addr1,
			srcContractIBCPort: "myIBCPort",
			srcMsg: wvm.CosmosMsg{
				IBC: &wvm.IBCMsg{
					Transfer: &wvm.TransferMsg{
						ChannelID: "myChanID",
						ToAddress: addr2.String(),
						Amount: wvm.Coin{
							Denom:  "ALX",
							Amount: "1",
						},
						Timeout: wvm.IBCTimeout{Timestamp: 100},
					},
				},
			},
			transferPortSource: wasmtesting.MockIBCTransferKeeper{GetPortFn: func(ctx sdk.Context) string {
				return "transfer"
			}},
			output: []sdk.Msg{
				&ibctransfertypes.MsgTransfer{
					SourcePort:    "transfer",
					SourceChannel: "myChanID",
					Token: sdk.Coin{
						Denom:  "ALX",
						Amount: sdk.NewInt(1),
					},
					Sender:           addr1.String(),
					Receiver:         addr2.String(),
					TimeoutTimestamp: 100,
				},
			},
		},
		"IBC transfer with time and height timeout": {
			sender:             addr1,
			srcContractIBCPort: "myIBCPort",
			srcMsg: wvm.CosmosMsg{
				IBC: &wvm.IBCMsg{
					Transfer: &wvm.TransferMsg{
						ChannelID: "myChanID",
						ToAddress: addr2.String(),
						Amount: wvm.Coin{
							Denom:  "ALX",
							Amount: "1",
						},
						Timeout: wvm.IBCTimeout{Timestamp: 100, Block: &wvm.IBCTimeoutBlock{Height: 1, Revision: 2}},
					},
				},
			},
			transferPortSource: wasmtesting.MockIBCTransferKeeper{GetPortFn: func(ctx sdk.Context) string {
				return "transfer"
			}},
			output: []sdk.Msg{
				&ibctransfertypes.MsgTransfer{
					SourcePort:    "transfer",
					SourceChannel: "myChanID",
					Token: sdk.Coin{
						Denom:  "ALX",
						Amount: sdk.NewInt(1),
					},
					Sender:           addr1.String(),
					Receiver:         addr2.String(),
					TimeoutTimestamp: 100,
					TimeoutHeight:    clienttypes.NewHeight(2, 1),
				},
			},
		},
		"IBC close channel": {
			sender:             addr1,
			srcContractIBCPort: "myIBCPort",
			srcMsg: wvm.CosmosMsg{
				IBC: &wvm.IBCMsg{
					CloseChannel: &wvm.CloseChannelMsg{
						ChannelID: "channel-1",
					},
				},
			},
			output: []sdk.Msg{
				&channeltypes.MsgChannelCloseInit{
					PortId:    "wasm." + addr1.String(),
					ChannelId: "channel-1",
					Signer:    addr1.String(),
				},
			},
		},
	}
	encodingConfig := MakeEncodingConfig(t)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var ctx sdk.Context
			encoder := DefaultEncoders(encodingConfig.Codec, tc.transferPortSource)
			res, err := encoder.Encode(ctx, tc.sender, tc.srcContractIBCPort, tc.srcMsg)
			if tc.expError {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.output, res)

			// and valid sdk message
			for _, v := range res {
				gotErr := v.ValidateBasic()
				if tc.expInvalid {
					assert.Error(t, gotErr)
				} else {
					assert.NoError(t, gotErr)
				}
			}
		})
	}
}

func TestEncodeGovMsg(t *testing.T) {
	myAddr := RandomAccountAddress(t)

	cases := map[string]struct {
		sender             sdk.AccAddress
		srcMsg             wvm.CosmosMsg
		transferPortSource types.ICS20TransferPortSource
		// set if valid
		output []sdk.Msg
		// set if expect mapping fails
		expError bool
		// set if sdk validate basic should fail
		expInvalid bool
	}{
		"Gov vote: yes": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					Vote: &wvm.VoteMsg{ProposalId: 1, Vote: wvm.Yes},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVote{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Option:     v1.OptionYes,
				},
			},
		},
		"Gov vote: No": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					Vote: &wvm.VoteMsg{ProposalId: 1, Vote: wvm.No},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVote{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Option:     v1.OptionNo,
				},
			},
		},
		"Gov vote: Abstain": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					Vote: &wvm.VoteMsg{ProposalId: 10, Vote: wvm.Abstain},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVote{
					ProposalId: 10,
					Voter:      myAddr.String(),
					Option:     v1.OptionAbstain,
				},
			},
		},
		"Gov vote: No with veto": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					Vote: &wvm.VoteMsg{ProposalId: 1, Vote: wvm.NoWithVeto},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVote{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Option:     v1.OptionNoWithVeto,
				},
			},
		},
		"Gov vote: unset option": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					Vote: &wvm.VoteMsg{ProposalId: 1},
				},
			},
			expError: true,
		},
		"Gov weighted vote: single vote": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					VoteWeighted: &wvm.VoteWeightedMsg{
						ProposalId: 1,
						Options: []wvm.WeightedVoteOption{
							{Option: wvm.Yes, Weight: "1"},
						},
					},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVoteWeighted{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Options: []*v1.WeightedVoteOption{
						{Option: v1.OptionYes, Weight: sdk.NewDec(1).String()},
					},
				},
			},
		},
		"Gov weighted vote: splitted": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					VoteWeighted: &wvm.VoteWeightedMsg{
						ProposalId: 1,
						Options: []wvm.WeightedVoteOption{
							{Option: wvm.Yes, Weight: "0.23"},
							{Option: wvm.No, Weight: "0.24"},
							{Option: wvm.Abstain, Weight: "0.26"},
							{Option: wvm.NoWithVeto, Weight: "0.27"},
						},
					},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVoteWeighted{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Options: []*v1.WeightedVoteOption{
						{Option: v1.OptionYes, Weight: sdk.NewDecWithPrec(23, 2).String()},
						{Option: v1.OptionNo, Weight: sdk.NewDecWithPrec(24, 2).String()},
						{Option: v1.OptionAbstain, Weight: sdk.NewDecWithPrec(26, 2).String()},
						{Option: v1.OptionNoWithVeto, Weight: sdk.NewDecWithPrec(27, 2).String()},
					},
				},
			},
		},
		"Gov weighted vote: duplicate option - invalid": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					VoteWeighted: &wvm.VoteWeightedMsg{
						ProposalId: 1,
						Options: []wvm.WeightedVoteOption{
							{Option: wvm.Yes, Weight: "0.5"},
							{Option: wvm.Yes, Weight: "0.5"},
						},
					},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVoteWeighted{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Options: []*v1.WeightedVoteOption{
						{Option: v1.OptionYes, Weight: sdk.NewDecWithPrec(5, 1).String()},
						{Option: v1.OptionYes, Weight: sdk.NewDecWithPrec(5, 1).String()},
					},
				},
			},
			expInvalid: true,
		},
		"Gov weighted vote: weight sum exceeds 1- invalid": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					VoteWeighted: &wvm.VoteWeightedMsg{
						ProposalId: 1,
						Options: []wvm.WeightedVoteOption{
							{Option: wvm.Yes, Weight: "0.51"},
							{Option: wvm.No, Weight: "0.5"},
						},
					},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVoteWeighted{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Options: []*v1.WeightedVoteOption{
						{Option: v1.OptionYes, Weight: sdk.NewDecWithPrec(51, 2).String()},
						{Option: v1.OptionNo, Weight: sdk.NewDecWithPrec(5, 1).String()},
					},
				},
			},
			expInvalid: true,
		},
		"Gov weighted vote: weight sum less than 1 - invalid": {
			sender: myAddr,
			srcMsg: wvm.CosmosMsg{
				Gov: &wvm.GovMsg{
					VoteWeighted: &wvm.VoteWeightedMsg{
						ProposalId: 1,
						Options: []wvm.WeightedVoteOption{
							{Option: wvm.Yes, Weight: "0.49"},
							{Option: wvm.No, Weight: "0.5"},
						},
					},
				},
			},
			output: []sdk.Msg{
				&v1.MsgVoteWeighted{
					ProposalId: 1,
					Voter:      myAddr.String(),
					Options: []*v1.WeightedVoteOption{
						{Option: v1.OptionYes, Weight: sdk.NewDecWithPrec(49, 2).String()},
						{Option: v1.OptionNo, Weight: sdk.NewDecWithPrec(5, 1).String()},
					},
				},
			},
			expInvalid: true,
		},
	}
	encodingConfig := MakeEncodingConfig(t)
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			var ctx sdk.Context
			encoder := DefaultEncoders(encodingConfig.Codec, tc.transferPortSource)
			res, gotEncErr := encoder.Encode(ctx, tc.sender, "myIBCPort", tc.srcMsg)
			if tc.expError {
				assert.Error(t, gotEncErr)
				return
			}
			require.NoError(t, gotEncErr)
			assert.Equal(t, tc.output, res)

			// and valid sdk message
			for _, v := range res {
				gotErr := v.ValidateBasic()
				if tc.expInvalid {
					assert.Error(t, gotErr)
				} else {
					assert.NoError(t, gotErr)
				}
			}
		})
	}
}

func TestConvertWasmCoinToSdkCoin(t *testing.T) {
	specs := map[string]struct {
		src    wvm.Coin
		expErr bool
		expVal sdk.Coin
	}{
		"all good": {
			src: wvm.Coin{
				Denom:  "foo",
				Amount: "1",
			},
			expVal: sdk.NewCoin("foo", sdk.NewIntFromUint64(1)),
		},
		"negative amount": {
			src: wvm.Coin{
				Denom:  "foo",
				Amount: "-1",
			},
			expErr: true,
		},
		"denom to short": {
			src: wvm.Coin{
				Denom:  "f",
				Amount: "1",
			},
			expErr: true,
		},
		"invalid demum char": {
			src: wvm.Coin{
				Denom:  "&fff",
				Amount: "1",
			},
			expErr: true,
		},
		"not a number amount": {
			src: wvm.Coin{
				Denom:  "foo",
				Amount: "bar",
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotVal, gotErr := ConvertWasmCoinToSdkCoin(spec.src)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.expVal, gotVal)
		})
	}
}

func TestConvertWasmCoinsToSdkCoins(t *testing.T) {
	specs := map[string]struct {
		src    []wvm.Coin
		exp    sdk.Coins
		expErr bool
	}{
		"empty": {
			src: []wvm.Coin{},
			exp: nil,
		},
		"single coin": {
			src: []wvm.Coin{{Denom: "foo", Amount: "1"}},
			exp: sdk.NewCoins(sdk.NewCoin("foo", sdk.NewInt(1))),
		},
		"multiple coins": {
			src: []wvm.Coin{
				{Denom: "foo", Amount: "1"},
				{Denom: "bar", Amount: "2"},
			},
			exp: sdk.NewCoins(
				sdk.NewCoin("bar", sdk.NewInt(2)),
				sdk.NewCoin("foo", sdk.NewInt(1)),
			),
		},
		"sorted": {
			src: []wvm.Coin{
				{Denom: "foo", Amount: "1"},
				{Denom: "other", Amount: "1"},
				{Denom: "bar", Amount: "1"},
			},
			exp: []sdk.Coin{
				sdk.NewCoin("bar", sdk.NewInt(1)),
				sdk.NewCoin("foo", sdk.NewInt(1)),
				sdk.NewCoin("other", sdk.NewInt(1)),
			},
		},
		"zero amounts dropped": {
			src: []wvm.Coin{
				{Denom: "foo", Amount: "1"},
				{Denom: "bar", Amount: "0"},
			},
			exp: sdk.NewCoins(
				sdk.NewCoin("foo", sdk.NewInt(1)),
			),
		},
		"duplicate denoms merged": {
			src: []wvm.Coin{
				{Denom: "foo", Amount: "1"},
				{Denom: "foo", Amount: "1"},
			},
			exp: []sdk.Coin{sdk.NewCoin("foo", sdk.NewInt(2))},
		},
		"duplicate denoms with one 0 amount does not fail": {
			src: []wvm.Coin{
				{Denom: "foo", Amount: "0"},
				{Denom: "foo", Amount: "1"},
			},
			exp: []sdk.Coin{sdk.NewCoin("foo", sdk.NewInt(1))},
		},
		"empty denom rejected": {
			src:    []wvm.Coin{{Denom: "", Amount: "1"}},
			expErr: true,
		},
		"invalid denom rejected": {
			src:    []wvm.Coin{{Denom: "!%&", Amount: "1"}},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			gotCoins, gotErr := ConvertWasmCoinsToSdkCoins(spec.src)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, spec.exp, gotCoins)
			assert.NoError(t, gotCoins.Validate())
		})
	}
}

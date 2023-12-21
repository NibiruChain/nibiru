package cmd

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	proto "github.com/cosmos/gogoproto/proto"
	"github.com/spf13/cobra"

	// stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/client"
	_ "github.com/cosmos/cosmos-sdk/client/docs/statik"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vestingtypes "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	crisistypes "github.com/cosmos/cosmos-sdk/x/crisis/types"
	evidencetypes "github.com/cosmos/cosmos-sdk/x/evidence/types"
	govtypesv1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	govtypesv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"

	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"

	feegranttypes "github.com/cosmos/cosmos-sdk/x/feegrant"
	grouptypes "github.com/cosmos/cosmos-sdk/x/group"
	slashingtypes "github.com/cosmos/cosmos-sdk/x/slashing/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	feetypes "github.com/cosmos/ibc-go/v7/modules/apps/29-fee/types"
	transfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcclient "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	connectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
	coretypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"

	wasmvm "github.com/CosmWasm/wasmvm/types"
	upgradetypes "github.com/cosmos/cosmos-sdk/x/upgrade/types"

	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
	oracletypes "github.com/NibiruChain/nibiru/x/oracle/types"
	perptypes "github.com/NibiruChain/nibiru/x/perp/v2/types"

	spottypes "github.com/NibiruChain/nibiru/x/spot/types"

	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"

	tokenfactorytypes "github.com/NibiruChain/nibiru/x/tokenfactory/types"
)

func MakeMap() map[string]proto.Message {
	return map[string]proto.Message{
		"/cosmos.auth.v1beta1.MsgUpdateParams":                        new(authtypes.MsgUpdateParams),
		"/cosmos.authz.v1beta1.MsgExec":                               new(authztypes.MsgExec),
		"/cosmos.authz.v1beta1.MsgGrant":                              new(authztypes.MsgGrant),
		"/cosmos.authz.v1beta1.MsgRevoke":                             new(authztypes.MsgRevoke),
		"/cosmos.bank.v1beta1.MsgMultiSend":                           new(banktypes.MsgMultiSend),
		"/cosmos.bank.v1beta1.MsgSend":                                new(banktypes.MsgSend),
		"/cosmos.bank.v1beta1.MsgSetSendEnabled":                      new(banktypes.MsgSetSendEnabled),
		"/cosmos.bank.v1beta1.MsgUpdateParams":                        new(banktypes.MsgUpdateParams),
		"/cosmos.crisis.v1beta1.MsgUpdateParams":                      new(crisistypes.MsgUpdateParams),
		"/cosmos.crisis.v1beta1.MsgVerifyInvariant":                   new(crisistypes.MsgVerifyInvariant),
		"/cosmos.distribution.v1beta1.MsgCommunityPoolSpend":          new(distributiontypes.MsgCommunityPoolSpend),
		"/cosmos.distribution.v1beta1.MsgFundCommunityPool":           new(distributiontypes.MsgFundCommunityPool),
		"/cosmos.distribution.v1beta1.MsgSetWithdrawAddress":          new(distributiontypes.MsgSetWithdrawAddress),
		"/cosmos.distribution.v1beta1.MsgUpdateParams":                new(distributiontypes.MsgUpdateParams),
		"/cosmos.distribution.v1beta1.MsgWithdrawDelegatorReward":     new(distributiontypes.MsgWithdrawDelegatorReward),
		"/cosmos.distribution.v1beta1.MsgWithdrawValidatorCommission": new(distributiontypes.MsgWithdrawValidatorCommission),
		"/cosmos.evidence.v1beta1.MsgSubmitEvidence":                  new(evidencetypes.MsgSubmitEvidence),
		"/cosmos.feegrant.v1beta1.MsgGrantAllowance":                  new(feegranttypes.MsgGrantAllowance),
		"/cosmos.feegrant.v1beta1.MsgRevokeAllowance":                 new(feegranttypes.MsgRevokeAllowance),
		"/cosmos.gov.v1.MsgDeposit":                                   new(govtypesv1.MsgDeposit),
		"/cosmos.gov.v1.MsgExecLegacyContent":                         new(govtypesv1.MsgExecLegacyContent),
		"/cosmos.gov.v1.MsgSubmitProposal":                            new(govtypesv1.MsgSubmitProposal),
		"/cosmos.gov.v1.MsgUpdateParams":                              new(govtypesv1.MsgUpdateParams),
		"/cosmos.gov.v1.MsgVote":                                      new(govtypesv1.MsgVote),
		"/cosmos.gov.v1.MsgVoteWeighted":                              new(govtypesv1.MsgVoteWeighted),
		"/cosmos.gov.v1beta1.MsgDeposit":                              new(govtypesv1beta1.MsgDeposit),
		"/cosmos.gov.v1beta1.MsgSubmitProposal":                       new(govtypesv1beta1.MsgSubmitProposal),
		"/cosmos.gov.v1beta1.MsgVote":                                 new(govtypesv1beta1.MsgVote),
		"/cosmos.gov.v1beta1.MsgVoteWeighted":                         new(govtypesv1beta1.MsgVoteWeighted),
		"/cosmos.group.v1.MsgCreateGroup":                             new(grouptypes.MsgCreateGroup),
		"/cosmos.group.v1.MsgCreateGroupPolicy":                       new(grouptypes.MsgCreateGroupPolicy),
		"/cosmos.group.v1.MsgCreateGroupWithPolicy":                   new(grouptypes.MsgCreateGroupWithPolicy),
		"/cosmos.group.v1.MsgExec":                                    new(grouptypes.MsgExec),
		"/cosmos.group.v1.MsgLeaveGroup":                              new(grouptypes.MsgLeaveGroup),
		"/cosmos.group.v1.MsgSubmitProposal":                          new(grouptypes.MsgSubmitProposal),
		"/cosmos.group.v1.MsgUpdateGroupAdmin":                        new(grouptypes.MsgUpdateGroupAdmin),
		"/cosmos.group.v1.MsgUpdateGroupMembers":                      new(grouptypes.MsgUpdateGroupMembers),
		"/cosmos.group.v1.MsgUpdateGroupMetadata":                     new(grouptypes.MsgUpdateGroupMetadata),
		"/cosmos.group.v1.MsgUpdateGroupPolicyAdmin":                  new(grouptypes.MsgUpdateGroupPolicyAdmin),
		"/cosmos.group.v1.MsgUpdateGroupPolicyDecisionPolicy":         new(grouptypes.MsgUpdateGroupPolicyDecisionPolicy),
		"/cosmos.group.v1.MsgUpdateGroupPolicyMetadata":               new(grouptypes.MsgUpdateGroupPolicyMetadata),
		"/cosmos.group.v1.MsgVote":                                    new(grouptypes.MsgVote),
		"/cosmos.group.v1.MsgWithdrawProposal":                        new(grouptypes.MsgWithdrawProposal),
		"/cosmos.slashing.v1beta1.MsgUnjail":                          new(slashingtypes.MsgUnjail),
		"/cosmos.slashing.v1beta1.MsgUpdateParams":                    new(slashingtypes.MsgUpdateParams),
		"/cosmos.staking.v1beta1.MsgBeginRedelegate":                  new(stakingtypes.MsgBeginRedelegate),
		"/cosmos.staking.v1beta1.MsgCancelUnbondingDelegation":        new(stakingtypes.MsgCancelUnbondingDelegation),
		"/cosmos.staking.v1beta1.MsgCreateValidator":                  new(stakingtypes.MsgCreateValidator),
		"/cosmos.staking.v1beta1.MsgDelegate":                         new(stakingtypes.MsgDelegate),
		"/cosmos.staking.v1beta1.MsgEditValidator":                    new(stakingtypes.MsgEditValidator),
		"/cosmos.staking.v1beta1.MsgUndelegate":                       new(stakingtypes.MsgUndelegate),
		"/cosmos.staking.v1beta1.MsgUpdateParams":                     new(stakingtypes.MsgUpdateParams),
		"/cosmos.upgrade.v1beta1.MsgCancelUpgrade":                    new(upgradetypes.MsgCancelUpgrade),
		"/cosmos.upgrade.v1beta1.MsgSoftwareUpgrade":                  new(upgradetypes.MsgSoftwareUpgrade),
		"/cosmos.vesting.v1beta1.MsgCreatePeriodicVestingAccount":     new(vestingtypes.MsgCreatePeriodicVestingAccount),
		"/cosmos.vesting.v1beta1.MsgCreatePermanentLockedAccount":     new(vestingtypes.MsgCreatePermanentLockedAccount),
		"/cosmos.vesting.v1beta1.MsgCreateVestingAccount":             new(vestingtypes.MsgCreateVestingAccount),
		"/cosmwasm.wasm.v1.MsgAddCodeUploadParamsAddresses":           new(wasmtypes.MsgAddCodeUploadParamsAddresses),
		"/cosmwasm.wasm.v1.MsgClearAdmin":                             new(wasmtypes.MsgClearAdmin),
		"/cosmwasm.wasm.v1.MsgExecuteContract":                        new(wasmtypes.MsgExecuteContract),
		"/cosmwasm.wasm.v1.MsgIBCCloseChannel":                        new(wasmtypes.MsgIBCCloseChannel),
		"/cosmwasm.wasm.v1.MsgIBCSend":                                new(wasmtypes.MsgIBCSend),
		"/cosmwasm.wasm.v1.MsgInstantiateContract":                    new(wasmtypes.MsgInstantiateContract),
		"/cosmwasm.wasm.v1.MsgInstantiateContract2":                   new(wasmtypes.MsgInstantiateContract2),
		"/cosmwasm.wasm.v1.MsgMigrateContract":                        new(wasmtypes.MsgMigrateContract),
		"/cosmwasm.wasm.v1.MsgPinCodes":                               new(wasmtypes.MsgPinCodes),
		"/cosmwasm.wasm.v1.MsgRemoveCodeUploadParamsAddresses":        new(wasmtypes.MsgRemoveCodeUploadParamsAddresses),
		"/cosmwasm.wasm.v1.MsgStoreAndInstantiateContract":            new(wasmtypes.MsgStoreAndInstantiateContract),
		"/cosmwasm.wasm.v1.MsgStoreAndMigrateContract":                new(wasmtypes.MsgStoreAndMigrateContract),
		"/cosmwasm.wasm.v1.MsgStoreCode":                              new(wasmtypes.MsgStoreCode),
		"/cosmwasm.wasm.v1.MsgSudoContract":                           new(wasmtypes.MsgSudoContract),
		"/cosmwasm.wasm.v1.MsgUnpinCodes":                             new(wasmtypes.MsgUnpinCodes),
		"/cosmwasm.wasm.v1.MsgUpdateAdmin":                            new(wasmtypes.MsgUpdateAdmin),
		"/cosmwasm.wasm.v1.MsgUpdateContractLabel":                    new(wasmtypes.MsgUpdateContractLabel),
		"/cosmwasm.wasm.v1.MsgUpdateInstantiateConfig":                new(wasmtypes.MsgUpdateInstantiateConfig),
		"/cosmwasm.wasm.v1.MsgUpdateParams":                           new(wasmtypes.MsgUpdateParams),
		"/ibc.applications.fee.v1.MsgPayPacketFee":                    new(feetypes.MsgPayPacketFee),
		"/ibc.applications.fee.v1.MsgPayPacketFeeAsync":               new(feetypes.MsgPayPacketFeeAsync),
		"/ibc.applications.fee.v1.MsgRegisterCounterpartyPayee":       new(feetypes.MsgRegisterCounterpartyPayee),
		"/ibc.applications.fee.v1.MsgRegisterPayee":                   new(feetypes.MsgRegisterPayee),
		"/ibc.applications.transfer.v1.MsgTransfer":                   new(transfertypes.MsgTransfer),
		"/ibc.core.channel.v1.MsgAcknowledgement":                     new(coretypes.MsgAcknowledgement),
		"/ibc.core.channel.v1.MsgChannelCloseConfirm":                 new(coretypes.MsgChannelCloseConfirm),
		"/ibc.core.channel.v1.MsgChannelCloseInit":                    new(coretypes.MsgChannelCloseInit),
		"/ibc.core.channel.v1.MsgChannelOpenAck":                      new(coretypes.MsgChannelOpenAck),
		"/ibc.core.channel.v1.MsgChannelOpenConfirm":                  new(coretypes.MsgChannelOpenConfirm),
		"/ibc.core.channel.v1.MsgChannelOpenInit":                     new(coretypes.MsgChannelOpenInit),
		"/ibc.core.channel.v1.MsgChannelOpenTry":                      new(coretypes.MsgChannelOpenTry),
		"/ibc.core.channel.v1.MsgRecvPacket":                          new(coretypes.MsgRecvPacket),
		"/ibc.core.channel.v1.MsgTimeout":                             new(coretypes.MsgTimeout),
		"/ibc.core.channel.v1.MsgTimeoutOnClose":                      new(coretypes.MsgTimeoutOnClose),
		"/ibc.core.client.v1.MsgCreateClient":                         new(ibcclient.MsgCreateClient),
		"/ibc.core.client.v1.MsgSubmitMisbehaviour":                   new(ibcclient.MsgSubmitMisbehaviour),
		"/ibc.core.client.v1.MsgUpdateClient":                         new(ibcclient.MsgUpdateClient),
		"/ibc.core.client.v1.MsgUpgradeClient":                        new(ibcclient.MsgUpgradeClient),
		"/ibc.core.connection.v1.MsgConnectionOpenAck":                new(connectiontypes.MsgConnectionOpenAck),
		"/ibc.core.connection.v1.MsgConnectionOpenConfirm":            new(connectiontypes.MsgConnectionOpenConfirm),
		"/ibc.core.connection.v1.MsgConnectionOpenInit":               new(connectiontypes.MsgConnectionOpenInit),
		"/ibc.core.connection.v1.MsgConnectionOpenTry":                new(connectiontypes.MsgConnectionOpenTry),
		"/nibiru.devgas.v1.MsgCancelFeeShare":                         new(devgastypes.MsgCancelFeeShare),
		"/nibiru.devgas.v1.MsgRegisterFeeShare":                       new(devgastypes.MsgRegisterFeeShare),
		"/nibiru.devgas.v1.MsgUpdateFeeShare":                         new(devgastypes.MsgUpdateFeeShare),
		"/nibiru.devgas.v1.MsgUpdateParams":                           new(devgastypes.MsgUpdateParams),
		"/nibiru.oracle.v1.MsgAggregateExchangeRatePrevote":           new(oracletypes.MsgAggregateExchangeRatePrevote),
		"/nibiru.oracle.v1.MsgAggregateExchangeRateVote":              new(oracletypes.MsgAggregateExchangeRateVote),
		"/nibiru.oracle.v1.MsgDelegateFeedConsent":                    new(oracletypes.MsgDelegateFeedConsent),
		"/nibiru.perp.v2.MsgAddMargin":                                new(perptypes.MsgAddMargin),
		"/nibiru.perp.v2.MsgAllocateEpochRebates":                     new(perptypes.MsgAllocateEpochRebates),
		"/nibiru.perp.v2.MsgChangeCollateralDenom":                    new(perptypes.MsgChangeCollateralDenom),
		"/nibiru.perp.v2.MsgClosePosition":                            new(perptypes.MsgClosePosition),
		"/nibiru.perp.v2.MsgDonateToEcosystemFund":                    new(perptypes.MsgDonateToEcosystemFund),
		"/nibiru.perp.v2.MsgMarketOrder":                              new(perptypes.MsgMarketOrder),
		"/nibiru.perp.v2.MsgMultiLiquidate":                           new(perptypes.MsgMultiLiquidate),
		"/nibiru.perp.v2.MsgPartialClose":                             new(perptypes.MsgPartialClose),
		"/nibiru.perp.v2.MsgRemoveMargin":                             new(perptypes.MsgRemoveMargin),
		"/nibiru.perp.v2.MsgSettlePosition":                           new(perptypes.MsgSettlePosition),
		"/nibiru.perp.v2.MsgShiftPegMultiplier":                       new(perptypes.MsgShiftPegMultiplier),
		"/nibiru.perp.v2.MsgShiftSwapInvariant":                       new(perptypes.MsgShiftSwapInvariant),
		"/nibiru.perp.v2.MsgWithdrawEpochRebates":                     new(perptypes.MsgWithdrawEpochRebates),
		"/nibiru.spot.v1.MsgCreatePool":                               new(spottypes.MsgCreatePool),
		"/nibiru.spot.v1.MsgExitPool":                                 new(spottypes.MsgExitPool),
		"/nibiru.spot.v1.MsgJoinPool":                                 new(spottypes.MsgJoinPool),
		"/nibiru.spot.v1.MsgSwapAssets":                               new(spottypes.MsgSwapAssets),
		"/nibiru.sudo.v1.MsgChangeRoot":                               new(sudotypes.MsgChangeRoot),
		"/nibiru.sudo.v1.MsgEditSudoers":                              new(sudotypes.MsgEditSudoers),
		"/nibiru.tokenfactory.v1.MsgBurn":                             new(tokenfactorytypes.MsgBurn),
		"/nibiru.tokenfactory.v1.MsgChangeAdmin":                      new(tokenfactorytypes.MsgChangeAdmin),
		"/nibiru.tokenfactory.v1.MsgCreateDenom":                      new(tokenfactorytypes.MsgCreateDenom),
		"/nibiru.tokenfactory.v1.MsgMint":                             new(tokenfactorytypes.MsgMint),
		"/nibiru.tokenfactory.v1.MsgSetDenomMetadata":                 new(tokenfactorytypes.MsgSetDenomMetadata),
		"/nibiru.tokenfactory.v1.MsgUpdateModuleParams":               new(tokenfactorytypes.MsgUpdateModuleParams),
	}
}

// YieldStargateMsgs parses the JSON and sends wasmvm.StargateMsg objects to a channel
func YieldStargateMsgs(jsonBz []byte) ([]wasmvm.StargateMsg, error) {
	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return nil, err
	}

	var msgs []wasmvm.StargateMsg
	parseStargateMsgs(data, &msgs)
	return msgs, nil
}

func parseStargateMsgs(jsonData any, msgs *[]wasmvm.StargateMsg) {
	switch v := jsonData.(type) {
	case map[string]interface{}:
		if typeURL, ok := v["type_url"].(string); ok {
			if value, ok := v["value"].(string); ok {
				*msgs = append(*msgs, wasmvm.StargateMsg{
					TypeURL: typeURL,
					Value:   []byte(value),
				})
			}
		}
		for _, value := range v {
			parseStargateMsgs(value, msgs)
		}
	case []interface{}:
		for _, value := range v {
			parseStargateMsgs(value, msgs)
		}
	}
}

type StargateMsgDecoded struct {
	TypeURL string `json:"type_url"`
	Value   string `json:"value"`
}

func DecodeBase64StargateMsgs(
	jsonBz []byte, codec codec.Codec,
) (newSgMsgs []StargateMsgDecoded, err error) {
	messageMap := MakeMap()

	var data interface{}
	if err := json.Unmarshal(jsonBz, &data); err != nil {
		return []StargateMsgDecoded{}, err
	}

	sgMsgs, err := YieldStargateMsgs(jsonBz)
	if err != nil {
		return
	}
	for _, sgMsg := range sgMsgs {
		valueStr := string(sgMsg.Value)
		value := strings.Replace(string(sgMsg.Value), `\"`, `"`, 0)
		value = strings.Replace(value, `"{`, `{`, 0)
		value = strings.Replace(value, `}"`, `}`, 0)

		if _, err := base64.StdEncoding.DecodeString(valueStr); err == nil {
			protoTypes := messageMap[sgMsg.TypeURL]

			decodedBz, _ := base64.StdEncoding.Strict().DecodeString(string(sgMsg.Value))
			concrete := protoTypes.(sdkcodec.ProtoMarshaler)

			err = codec.Unmarshal(decodedBz, concrete)
			if err != nil {
				return newSgMsgs, err
			}

			outBytes, err := codec.MarshalJSON(concrete)
			if err != nil {
				return newSgMsgs, err
			}

			newSgMsgs = append(newSgMsgs, StargateMsgDecoded{sgMsg.TypeURL, string(outBytes)})
		} else if _, err := json.Marshal(value); err == nil {
			newSgMsgs = append(newSgMsgs, StargateMsgDecoded{sgMsg.TypeURL, string(sgMsg.Value)})
		} else {
			return newSgMsgs, fmt.Errorf(
				"parse error: encountered wasmvm.StargateMsg with unexpected format: %s", sgMsg)
		}
	}
	return newSgMsgs, nil
}

// DecodeBase64Cmd creates a cobra command for base64 decoding.
func DecodeBase64Cmd(defaultNodeHome string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "base64-decode",
		Short: "Decode a base64-encoded protobuf message",
		Long: `Decode a base64-encoded protobuf message from JSON input.
The input should be a JSON object with 'type_url' and 'value' fields.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			clientCtx := client.GetClientContextFromCmd(cmd)

			outMessage, err := DecodeBase64StargateMsgs([]byte(args[0]), clientCtx.Codec)
			fmt.Println(outMessage)

			return err
		},
	}

	cmd.Flags().String(flags.FlagHome, defaultNodeHome, "The application home directory")

	return cmd
}

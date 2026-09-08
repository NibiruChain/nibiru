//! Implements the prost::Name trait for Nibiru protobuf types, which defines
//! the prost::Message.type_url function needed for CosmWasm smart contracts.

use prost::Name;

use crate::proto::{eth, nibiru};

const PACKAGE_TOKENFACTORY: &str = "nibiru.tokenfactory.v1";
const PACKAGE_ORACLE: &str = "nibiru.oracle.v1";
const PACKAGE_EPOCHS: &str = "nibiru.epochs.v1";
const PACKAGE_SPOT: &str = "nibiru.spot.v1";
const PACKAGE_PERP: &str = "nibiru.perp.v2";
const PACKAGE_INFLATION: &str = "nibiru.inflation.v1";
const PACKAGE_DEVGAS: &str = "nibiru.devgas.v1";
const PACKAGE_SUDO: &str = "nibiru.sudo.v1";
const PACKAGE_EVM: &str = "eth.evm.v1";

// TOKENFACTORY tx msg

impl Name for nibiru::tokenfactory::MsgCreateDenom {
    const NAME: &'static str = "MsgCreateDenom";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::MsgChangeAdmin {
    const NAME: &'static str = "MsgChangeAdmin";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::MsgUpdateModuleParams {
    const NAME: &'static str = "MsgUpdateModuleParams";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::MsgMint {
    const NAME: &'static str = "MsgMint";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::MsgBurn {
    const NAME: &'static str = "MsgBurn";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::MsgSetDenomMetadata {
    const NAME: &'static str = "MsgSetDenomMetadata";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}

// TOKENFACTORY query

impl Name for nibiru::tokenfactory::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::QueryDenomsRequest {
    const NAME: &'static str = "QueryDenomsRequest";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}
impl Name for nibiru::tokenfactory::QueryDenomInfoRequest {
    const NAME: &'static str = "QueryDenomInfoRequest";
    const PACKAGE: &'static str = PACKAGE_TOKENFACTORY;
}

// EPOCHS query

impl Name for nibiru::epochs::QueryEpochInfosRequest {
    const NAME: &'static str = "QueryEpochInfosRequest";
    const PACKAGE: &'static str = PACKAGE_EPOCHS;
}
impl Name for nibiru::epochs::QueryCurrentEpochRequest {
    const NAME: &'static str = "QueryCurrentEpochRequest";
    const PACKAGE: &'static str = PACKAGE_EPOCHS;
}

// ORACLE tx msg

impl Name for nibiru::oracle::MsgEditOracleParams {
    const NAME: &'static str = "MsgEditOracleParams";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}

// ORACLE query

impl Name for nibiru::oracle::QueryExchangeRateRequest {
    const NAME: &'static str = "QueryExchangeRateRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
// TODO: This type exists but was not exported by the protos. Why?
// impl Name for nibiru::oracle::QueryExchangeRateTwapRequest {
//     const NAME: &'static str = "QueryExchangeRateTwapRequest";
//     const PACKAGE: &'static str = PACKAGE_ORACLE;
// }
impl Name for nibiru::oracle::QueryExchangeRatesRequest {
    const NAME: &'static str = "QueryExchangeRatesRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryActivesRequest {
    const NAME: &'static str = "QueryActivesRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryVoteTargetsRequest {
    const NAME: &'static str = "QueryVoteTargetsRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryFeederDelegationRequest {
    const NAME: &'static str = "QueryFeederDelegationRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryMissCounterRequest {
    const NAME: &'static str = "QueryMissCounterRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryAggregatePrevoteRequest {
    const NAME: &'static str = "QueryAggregatePrevoteRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryAggregatePrevotesRequest {
    const NAME: &'static str = "QueryAggregatePrevotesRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryAggregateVoteRequest {
    const NAME: &'static str = "QueryAggregateVoteRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryAggregateVotesRequest {
    const NAME: &'static str = "QueryAggregateVotesRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}
impl Name for nibiru::oracle::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_ORACLE;
}

// SPOT query

impl Name for nibiru::spot::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryPoolNumberRequest {
    const NAME: &'static str = "QueryPoolNumberRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryPoolRequest {
    const NAME: &'static str = "QueryPoolRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryPoolsRequest {
    const NAME: &'static str = "QueryPoolsRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryPoolParamsRequest {
    const NAME: &'static str = "QueryPoolParamsRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryNumPoolsRequest {
    const NAME: &'static str = "QueryNumPoolsRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryTotalLiquidityRequest {
    const NAME: &'static str = "QueryTotalLiquidityRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryTotalPoolLiquidityRequest {
    const NAME: &'static str = "QueryTotalPoolLiquidityRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryTotalSharesRequest {
    const NAME: &'static str = "QueryTotalSharesRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QuerySpotPriceRequest {
    const NAME: &'static str = "QuerySpotPriceRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QuerySwapExactAmountInRequest {
    const NAME: &'static str = "QuerySwapExactAmountInRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QuerySwapExactAmountOutRequest {
    const NAME: &'static str = "QuerySwapExactAmountOutRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryJoinExactAmountInRequest {
    const NAME: &'static str = "QueryJoinExactAmountInRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryJoinExactAmountOutRequest {
    const NAME: &'static str = "QueryJoinExactAmountOutRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryExitExactAmountInRequest {
    const NAME: &'static str = "QueryExitExactAmountInRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::QueryExitExactAmountOutRequest {
    const NAME: &'static str = "QueryExitExactAmountOutRequest";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}

// SPOT tx msg

impl Name for nibiru::spot::MsgCreatePool {
    const NAME: &'static str = "MsgCreatePool";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::MsgJoinPool {
    const NAME: &'static str = "MsgJoinPool";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::MsgExitPool {
    const NAME: &'static str = "MsgExitPool";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}
impl Name for nibiru::spot::MsgSwapAssets {
    const NAME: &'static str = "MsgSwapAssets";
    const PACKAGE: &'static str = PACKAGE_SPOT;
}

// PERP tx msg

impl Name for nibiru::perp::MsgRemoveMargin {
    const NAME: &'static str = "MsgRemoveMargin";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgAddMargin {
    const NAME: &'static str = "MsgAddMargin";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgMultiLiquidate {
    const NAME: &'static str = "MsgMultiLiquidate";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgMarketOrder {
    const NAME: &'static str = "MsgMarketOrder";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgClosePosition {
    const NAME: &'static str = "MsgClosePosition";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgPartialClose {
    const NAME: &'static str = "MsgPartialClose";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgDonateToEcosystemFund {
    const NAME: &'static str = "MsgDonateToEcosystemFund";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgSettlePosition {
    const NAME: &'static str = "MsgSettlePosition";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgChangeCollateralDenom {
    const NAME: &'static str = "MsgChangeCollateralDenom";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgAllocateEpochRebates {
    const NAME: &'static str = "MsgAllocateEpochRebates";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgWithdrawEpochRebates {
    const NAME: &'static str = "MsgWithdrawEpochRebates";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgShiftPegMultiplier {
    const NAME: &'static str = "MsgShiftPegMultiplier";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgShiftSwapInvariant {
    const NAME: &'static str = "MsgShiftSwapInvariant";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgWithdrawFromPerpFund {
    const NAME: &'static str = "MsgWithdrawFromPerpFund";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::MsgCloseMarket {
    const NAME: &'static str = "MsgCloseMarket";
    const PACKAGE: &'static str = PACKAGE_PERP;
}

// PERP query

impl Name for nibiru::perp::QueryPositionRequest {
    const NAME: &'static str = "QueryPositionRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::QueryPositionsRequest {
    const NAME: &'static str = "QueryPositionsRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::QueryPositionStoreRequest {
    const NAME: &'static str = "QueryPositionStoreRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::QueryModuleAccountsRequest {
    const NAME: &'static str = "QueryModuleAccountsRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::QueryMarketsRequest {
    const NAME: &'static str = "QueryMarketsRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}
impl Name for nibiru::perp::QueryCollateralRequest {
    const NAME: &'static str = "QueryCollateralRequest";
    const PACKAGE: &'static str = PACKAGE_PERP;
}

// INFLATION query

impl Name for nibiru::inflation::QueryPeriodRequest {
    const NAME: &'static str = "QueryPeriodRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}
impl Name for nibiru::inflation::QueryEpochMintProvisionRequest {
    const NAME: &'static str = "QueryEpochMintProvisionRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}
impl Name for nibiru::inflation::QuerySkippedEpochsRequest {
    const NAME: &'static str = "QuerySkippedEpochsRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}
impl Name for nibiru::inflation::QueryCirculatingSupplyRequest {
    const NAME: &'static str = "QueryCirculatingSupplyRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}
impl Name for nibiru::inflation::QueryInflationRateRequest {
    const NAME: &'static str = "QueryInflationRateRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}
impl Name for nibiru::inflation::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_INFLATION;
}

// DEVGAS tx msg

impl Name for nibiru::devgas::MsgRegisterFeeShare {
    const NAME: &'static str = "MsgRegisterFeeShare";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::MsgUpdateFeeShare {
    const NAME: &'static str = "MsgUpdateFeeShare";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::MsgCancelFeeShare {
    const NAME: &'static str = "MsgCancelFeeShare";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::MsgUpdateParams {
    const NAME: &'static str = "MsgUpdateParams";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}

// DEVGAS query

impl Name for nibiru::devgas::QueryFeeSharesRequest {
    const NAME: &'static str = "QueryFeeSharesRequest";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::QueryFeeShareRequest {
    const NAME: &'static str = "QueryFeeShareRequest";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}
impl Name for nibiru::devgas::QueryFeeSharesByWithdrawerRequest {
    const NAME: &'static str = "QueryFeeSharesByWithdrawerRequest";
    const PACKAGE: &'static str = PACKAGE_DEVGAS;
}

// SUDO tx msg

impl Name for nibiru::sudo::MsgEditSudoers {
    const NAME: &'static str = "MsgEditSudoers";
    const PACKAGE: &'static str = PACKAGE_SUDO;
}
impl Name for nibiru::sudo::MsgChangeRoot {
    const NAME: &'static str = "MsgChangeRoot";
    const PACKAGE: &'static str = PACKAGE_SUDO;
}

// SUDO query

impl Name for nibiru::sudo::QuerySudoersRequest {
    const NAME: &'static str = "QuerySudoersRequest";
    const PACKAGE: &'static str = PACKAGE_SUDO;
}

// EVM tx msg

impl Name for eth::evm::MsgEthereumTx {
    const NAME: &'static str = "MsgEthereumTx";
    const PACKAGE: &'static str = PACKAGE_EVM;
}
impl Name for eth::evm::MsgUpdateParams {
    const NAME: &'static str = "MsgUpdateParams";
    const PACKAGE: &'static str = PACKAGE_EVM;
}
impl Name for eth::evm::MsgCreateFunToken {
    const NAME: &'static str = "MsgCreateFunToken";
    const PACKAGE: &'static str = PACKAGE_EVM;
}
impl Name for eth::evm::MsgConvertCoinToEvm {
    const NAME: &'static str = "MsgConvertCoinToEvm";
    const PACKAGE: &'static str = PACKAGE_EVM;
}

#[cfg(test)]
pub mod tests {

    use crate::{
        errors::TestResult,
        proto::{
            cosmos, eth,
            nibiru::{self},
            NibiruProstMsg, NibiruStargateMsg, NibiruStargateQuery,
        },
    };

    use cosmwasm_std as cw;

    #[test]
    fn stargate_tokenfactory_msgs() -> TestResult {
        let test_cases: Vec<(&str, cw::CosmosMsg)> = vec![
            (
                "/nibiru.tokenfactory.v1.MsgMint",
                nibiru::tokenfactory::MsgMint::default().into_stargate_msg(),
            ),
            (
                "/nibiru.tokenfactory.v1.MsgBurn",
                nibiru::tokenfactory::MsgBurn::default().into_stargate_msg(),
            ),
            (
                "/nibiru.tokenfactory.v1.MsgChangeAdmin",
                nibiru::tokenfactory::MsgChangeAdmin::default()
                    .into_stargate_msg(),
            ),
            (
                "/nibiru.tokenfactory.v1.MsgSetDenomMetadata",
                nibiru::tokenfactory::MsgSetDenomMetadata::default()
                    .into_stargate_msg(),
            ),
            (
                "/nibiru.tokenfactory.v1.MsgUpdateModuleParams",
                nibiru::tokenfactory::MsgUpdateModuleParams::default()
                    .into_stargate_msg(),
            ),
        ];

        for test_case in test_cases {
            let (tc_type_url, stargate_msg) = test_case;
            if let cw::CosmosMsg::Stargate {
                type_url,
                value: _value,
            } = stargate_msg.clone()
            {
                assert_eq!(tc_type_url, type_url)
            } else {
                panic!(
                    "Expected CosmosMsg::Stargate from CosmosMsg: {:#?}",
                    stargate_msg
                )
            }
        }

        println!(
            "prost::Name corresponding to a CosmosMsg should error if we \
            try converting it to QueryRequest::Stargate"
        );
        let pb_msg = nibiru::tokenfactory::MsgSetDenomMetadata::default();
        pb_msg
            .into_stargate_query()
            .expect_err("query is not a Msg");

        Ok(())
    }

    /// Uses values produced from the chain's protobuf marshaler to verify that
    /// our `nibiru-std` types encode the same way.
    ///
    /// ```
    /// // For example, one test case came from:
    /// fmt.Printf("%v\n", encodingConfig.Marshaler.MustMarshal(&tokenfactorytypes.MsgCreateDenom{
    ///     Sender:   "sender",
    ///     Subdenom: "subdenom",
    /// }))
    /// // which outputs "[10 6 115 101 110 100 101 114 18 8 115 117 98 100 101 110 111 109]"
    /// ```
    #[test]
    fn stargate_encoding() -> TestResult {
        let test_cases: Vec<(Box<dyn NibiruProstMsg>, Vec<u8>)> = vec![
            (
                Box::new(nibiru::tokenfactory::MsgCreateDenom {
                            sender: String::from("sender"),
                            subdenom: String::from("subdenom"),
                        }),
                parse_byte_string(
                            "[10 6 115 101 110 100 101 114 18 8 115 117 98 100 101 110 111 109]",
                        ),
            ),
            (
                Box::new(nibiru::tokenfactory::MsgMint {
                    sender: String::from("sender"),
                    coin: Some(cosmos::base::v1beta1::Coin { denom: String::from("abcxyz"), amount: String::from("123") }),
                    mint_to: String::from("mint_to") }),
                vec![10u8, 6, 115, 101, 110, 100, 101, 114, 18, 13, 10, 6, 97, 98, 99, 120, 121, 122, 18, 3, 49, 50, 51, 26, 7, 109, 105, 110, 116, 95, 116, 111]
            ),
        ];

        for (pb_msg, want_bz) in &test_cases {
            println!("pb_msg {{ value: {:?} }}", pb_msg.to_bytes(),);
            assert_eq!(*want_bz, pb_msg.to_bytes(), "pb_msg: {pb_msg:?}");
        }

        Ok(())
    }

    #[test]
    fn stargate_evm_msgs() -> TestResult {
        let test_cases: Vec<(&str, cw::CosmosMsg)> = vec![
            (
                "/eth.evm.v1.MsgEthereumTx",
                eth::evm::MsgEthereumTx::default().into_stargate_msg(),
            ),
            (
                "/eth.evm.v1.MsgUpdateParams",
                eth::evm::MsgUpdateParams::default().into_stargate_msg(),
            ),
            (
                "/eth.evm.v1.MsgCreateFunToken",
                eth::evm::MsgCreateFunToken::default().into_stargate_msg(),
            ),
            (
                "/eth.evm.v1.MsgConvertCoinToEvm",
                eth::evm::MsgConvertCoinToEvm::default().into_stargate_msg(),
            ),
        ];

        for test_case in test_cases {
            let (tc_type_url, stargate_msg) = test_case;
            if let cw::CosmosMsg::Stargate {
                type_url,
                value: _value,
            } = stargate_msg.clone()
            {
                assert_eq!(tc_type_url, type_url)
            } else {
                panic!(
                    "Expected CosmosMsg::Stargate from CosmosMsg: {:#?}",
                    stargate_msg
                )
            }
        }

        Ok(())
    }

    fn parse_byte_string(s: &str) -> Vec<u8> {
        s.trim_start_matches('[')
            .trim_end_matches(']')
            .split_whitespace()
            .filter_map(|byte| byte.parse::<u8>().ok())
            .collect()
    }
}

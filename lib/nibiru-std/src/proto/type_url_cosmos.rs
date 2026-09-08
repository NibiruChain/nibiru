//! Implements the prost::Name trait for cosmos protobuf types, which defines
//! the prost::Message.type_url function needed for CosmWasm smart contracts.

use prost::Name;

use crate::proto::cosmos;

const PACKAGE_BANK: &str = "cosmos.bank.v1beta1";
const PACKAGE_AUTH: &str = "cosmos.auth.v1beta1";
const PACKAGE_GOV: &str = "cosmos.gov.v1";

// BANK tx msg

impl Name for cosmos::bank::v1beta1::MsgSend {
    const NAME: &'static str = "MsgSend";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::MsgMultiSend {
    const NAME: &'static str = "MsgMultiSend";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::MsgUpdateParams {
    const NAME: &'static str = "MsgUpdateParams";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::MsgSetSendEnabled {
    const NAME: &'static str = "MsgSetSendEnabled";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

// BANK query

impl Name for cosmos::bank::v1beta1::QuerySupplyOfRequest {
    const NAME: &'static str = "QuerySupplyOfRequest";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::QueryBalanceRequest {
    const NAME: &'static str = "QueryBalanceRequest";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::QueryAllBalancesRequest {
    const NAME: &'static str = "QueryAllBalancesRequest";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

impl Name for cosmos::bank::v1beta1::QueryDenomMetadataRequest {
    const NAME: &'static str = "QueryDenomMetadataRequest";
    const PACKAGE: &'static str = PACKAGE_BANK;
}

// AUTH tx msg

impl Name for cosmos::auth::v1beta1::MsgUpdateParams {
    const NAME: &'static str = "MsgUpdateParams";
    const PACKAGE: &'static str = PACKAGE_AUTH;
}

// AUTH query

impl Name for cosmos::auth::v1beta1::QueryAccountInfoRequest {
    const NAME: &'static str = "QueryAccountInfoRequest";
    const PACKAGE: &'static str = PACKAGE_AUTH;
}

impl Name for cosmos::auth::v1beta1::QueryAccountRequest {
    const NAME: &'static str = "QueryAccountRequest";
    const PACKAGE: &'static str = PACKAGE_AUTH;
}

impl Name for cosmos::auth::v1beta1::QueryModuleAccountsRequest {
    const NAME: &'static str = "QueryModuleAccountsRequest";
    const PACKAGE: &'static str = PACKAGE_AUTH;
}

impl Name for cosmos::auth::v1beta1::QueryModuleAccountByNameRequest {
    const NAME: &'static str = "QueryModuleAccountByNameRequest";
    const PACKAGE: &'static str = PACKAGE_AUTH;
}

// GOV tx msg

impl Name for cosmos::gov::v1::MsgSubmitProposal {
    const NAME: &'static str = "MsgSubmitProposal";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::MsgExecLegacyContent {
    const NAME: &'static str = "MsgExecLegacyContent";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::MsgVote {
    const NAME: &'static str = "MsgVote";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::MsgVoteWeighted {
    const NAME: &'static str = "MsgVoteWeighted";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::MsgDeposit {
    const NAME: &'static str = "MsgDeposit";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::MsgUpdateParams {
    const NAME: &'static str = "MsgUpdateParams";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

// GOV query

impl Name for cosmos::gov::v1::QueryProposalRequest {
    const NAME: &'static str = "QueryProposalRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryProposalsRequest {
    const NAME: &'static str = "QueryProposalsRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryVoteRequest {
    const NAME: &'static str = "QueryVoteRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryVotesRequest {
    const NAME: &'static str = "QueryVotesRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryParamsRequest {
    const NAME: &'static str = "QueryParamsRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryDepositRequest {
    const NAME: &'static str = "QueryDepositRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryDepositsRequest {
    const NAME: &'static str = "QueryDepositsRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

impl Name for cosmos::gov::v1::QueryTallyResultRequest {
    const NAME: &'static str = "QueryTallyResultRequest";
    const PACKAGE: &'static str = PACKAGE_GOV;
}

#[cfg(test)]
mod tests {

    use cosmwasm_std::{Empty, QueryRequest};

    use crate::{
        errors::{NibiruResult, TestResult},
        proto::{cosmos, NibiruStargateQuery},
    };

    #[test]
    fn stargate_query_conversion() -> TestResult {
        let test_cases: Vec<(&str, NibiruResult<QueryRequest<Empty>>)> = vec![
            (
                "/cosmos.bank.v1beta1.Query/SupplyOf",
                cosmos::bank::v1beta1::QuerySupplyOfRequest {
                    denom: String::from("some_denom"),
                }
                .into_stargate_query(),
            ),
            (
                "/cosmos.bank.v1beta1.Query/Balance",
                cosmos::bank::v1beta1::QueryBalanceRequest {
                    address: String::from("some_address"),
                    denom: String::from("some_denom"),
                }
                .into_stargate_query(),
            ),
            (
                "/cosmos.bank.v1beta1.Query/DenomMetadata",
                cosmos::bank::v1beta1::QueryDenomMetadataRequest {
                    denom: String::from("some_denom"),
                }
                .into_stargate_query(),
            ),
        ];

        for test_case in test_cases {
            let test_case_path = test_case.0;
            let pb_query = test_case.1?;
            if let QueryRequest::Stargate { path, data: _data } = &pb_query {
                assert_eq!(test_case_path, path)
            } else {
                panic!("failed test on case: {:#?} ", test_case_path)
            }
        }

        Ok(())
    }
}

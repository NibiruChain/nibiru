//! nibiru-std::proto - traits.rs : Implements extensions for prost::Message
//! types for easy conversion to types needed for CosmWasm smart contracts.

// Allow deprecated variant `cosmwasm_std::CosmosMsg::Stargate` for compatibility
// with CosmWasm v1. Once we upgrade everything to v2 on Nibiru, we can remove
// this deprecate statement.
#![allow(deprecated)]
// TODO: remove allow(deprevated) ↑

use cosmwasm_std::{
    to_json_vec, Binary, ContractResult, CosmosMsg, CustomQuery, QuerierWrapper,
    QueryRequest, StdError, StdResult, SystemResult,
};

use crate::errors::{NibiruError, NibiruResult};

use crate::proto::cosmos;

pub trait NibiruProstMsg: prost::Message {
    /// Serialize this protobuf message as a byte vector
    fn to_bytes(&self) -> Vec<u8>;
    fn to_binary(&self) -> Binary;
    /// A type implementing prost::Message is not guaranteed to implement
    /// prost::Name and have a `Name.type_url()` function. This method attempts
    /// to downcast the message to prost::Name, and if successful, constructs a
    /// `CosmosMsg::Stargate` object corresponding to the type.
    ///
    /// This is commonly used when the protobuf type does not implement
    /// `prost::Name`, so the type URL must be provided explicitly.
    ///
    /// ```rust
    /// use cosmwasm_std::CosmosMsg;
    /// use nibiru_std::proto::{self, NibiruProstMsg};
    ///
    /// let bank_coin = proto::cosmos::base::v1beta1::Coin {
    ///     denom: "unibi".to_string(),
    ///     amount: "42".to_string(),
    /// };
    ///
    /// let msg = proto::eth::evm::MsgConvertCoinToEvm {
    ///     sender: "nibi1contractaddr".to_string(),
    ///     to_eth_addr: "0x000000000000000000000000000000000000dEaD".to_string(),
    ///     bank_coin: Some(bank_coin),
    /// };
    ///
    /// let stargate_msg =
    ///     msg.try_into_stargate_msg("/eth.evm.v1.MsgConvertCoinToEvm");
    ///
    /// if let CosmosMsg::Stargate { type_url, .. } = stargate_msg {
    ///     assert_eq!(type_url, "/eth.evm.v1.MsgConvertCoinToEvm");
    /// } else {
    ///     panic!("Expected CosmosMsg::Stargate variant");
    /// }
    /// ```
    fn try_into_stargate_msg(&self, type_url: &str) -> CosmosMsg {
        let value = self.to_binary();
        CosmosMsg::Stargate {
            type_url: type_url.to_string(),
            value,
        }
    }

    /// Parse into this protobuf type from `prost_types::Any`.
    fn from_any(any: &prost_types::Any) -> Result<Self, prost::DecodeError>
    where
        Self: Default + prost::Name + Sized,
    {
        any.to_msg()
    }
}

impl<M> NibiruProstMsg for M
where
    M: prost::Message,
{
    fn to_bytes(&self) -> Vec<u8> {
        self.encode_to_vec()
    }

    fn to_binary(&self) -> Binary {
        Binary::from(self.encode_to_vec())
    }
}

pub trait NibiruStargateMsg: prost::Message + prost::Name {
    #![allow(clippy::wrong_self_convention)]
    fn into_stargate_msg(&self) -> CosmosMsg;

    fn type_url(&self) -> String;
}

impl<M> NibiruStargateMsg for M
where
    M: prost::Message + prost::Name,
{
    /// Returns the `prost::Message` as a `CosmosMsg::Stargate` object.
    ///
    /// ```rust
    /// use cosmwasm_std::CosmosMsg;
    /// use nibiru_std::proto::{self, NibiruStargateMsg};
    ///
    /// let msg = proto::nibiru::tokenfactory::MsgMint {
    ///     sender: "nibi1sender".to_string(),
    ///     mint_to: "nibi1recipient".to_string(),
    ///     coin: Some(proto::cosmos::base::v1beta1::Coin {
    ///         denom: "unibi".to_string(),
    ///         amount: "7".to_string(),
    ///     }),
    /// };
    ///
    /// let stargate_msg = msg.into_stargate_msg();
    /// if let CosmosMsg::Stargate { type_url, .. } = stargate_msg {
    ///     assert_eq!(type_url, "/nibiru.tokenfactory.v1.MsgMint");
    /// } else {
    ///     panic!("Expected CosmosMsg::Stargate variant");
    /// }
    /// ```
    fn into_stargate_msg(&self) -> CosmosMsg {
        CosmosMsg::Stargate {
            type_url: self.type_url(),
            value: self.to_binary(),
        }
    }

    /// The "type URL" in the context of protobuf is used with a feature
    /// called "Any", a type that allows one to serialize and embed proto
    /// message (prost::Message) objects without as opaque values without having
    /// to predefine the type in the original message declaration.
    ///
    /// For example, a protobuf definition like:
    /// ```proto
    /// message CustomProtoMsg { string name = 1; }
    /// ```
    /// might have a type URL like "googleapis.com/package.name.CustomProtoMsg".
    /// Usage of `Any` with type URLs enables dynamic message composition and
    /// flexibility.
    ///
    /// We use these type URLs in CosmWasm and the Cosmos-SDK to classify
    /// gRPC messages for transactions and queries because Tendermint ABCI
    /// messages are protobuf objects.
    fn type_url(&self) -> String {
        format!("/{}.{}", Self::PACKAGE, Self::NAME)
    }
}

pub trait NibiruStargateQuery: prost::Message + prost::Name {
    #![allow(clippy::wrong_self_convention)]
    fn into_stargate_query(
        &self,
    ) -> NibiruResult<QueryRequest<cosmwasm_std::Empty>>;

    fn path(&self) -> String;
}

impl<M> NibiruStargateQuery for M
where
    M: prost::Message + prost::Name,
{
    /// Returns the `prost::Message` as a `QueryRequest::Stargate` object.
    /// Errors if the `prost::Name::type_url` does not indicate the type is a
    /// query.
    ///
    /// ```rust
    /// use cosmwasm_std::{Empty, QueryRequest};
    /// use nibiru_std::proto::{cosmos, NibiruStargateQuery};
    ///
    /// let query = cosmos::bank::v1beta1::QuerySupplyOfRequest {
    ///     denom: "unibi".to_string(),
    /// };
    ///
    /// let stargate_query: QueryRequest<Empty> =
    ///     query.into_stargate_query().expect("query conversion should work");
    /// if let QueryRequest::Stargate { path, .. } = stargate_query {
    ///     assert_eq!(path, "/cosmos.bank.v1beta1.Query/SupplyOf");
    /// } else {
    ///     panic!("Expected QueryRequest::Stargate variant");
    /// }
    /// ```
    fn into_stargate_query(
        &self,
    ) -> NibiruResult<QueryRequest<cosmwasm_std::Empty>> {
        if !self.type_url().contains("Query") {
            return Err(NibiruError::ProstNameisNotQuery {
                type_url: self.type_url(),
            });
        }
        Ok(QueryRequest::Stargate {
            path: self.path(),
            data: self.to_binary(),
        })
    }

    /// Fully qualified gRPC service path used for routing.
    /// Ex.: "/cosmos.bank.v1beta1.Query/SupplyOf"
    fn path(&self) -> String {
        let service_name = format!(
            "Query/{}",
            Self::NAME
                .trim_start_matches("Query")
                .trim_end_matches("Request")
        );
        format!("/{}.{}", Self::PACKAGE, service_name)
    }
}

/// Runs a Stargate query and decodes the protobuf response into a strong type.
///
/// `QuerierWrapper::query` decodes responses with serde JSON. Stargate query
/// responses are protobuf bytes, so callers need the lower-level `raw_query`
/// path followed by `prost::Message::decode`.
///
/// Contract usage should let Rust infer the request and querier types:
///
/// ```rust
/// use cosmwasm_std::{Deps, StdResult};
/// use nibiru_std::proto::{
///     cosmos::bank::v1beta1::{QueryBalanceRequest, QueryBalanceResponse},
///     query_stargate_proto,
/// };
///
/// pub fn query_bank_balance(
///     deps: Deps,
///     address: String,
///     denom: String,
/// ) -> StdResult<QueryBalanceResponse> {
///     let req = QueryBalanceRequest { address, denom };
///     let resp: QueryBalanceResponse = query_stargate_proto(&deps.querier, &req)?;
///     Ok(resp)
/// }
/// ```
pub fn query_stargate_proto<C, Req, Resp>(
    querier: &QuerierWrapper<C>,
    req: &Req,
) -> StdResult<Resp>
where
    C: CustomQuery,
    Req: NibiruStargateQuery,
    Resp: prost::Message + Default,
{
    let query = req.into_stargate_query()?;
    let raw_query = to_json_vec(&query)?;

    let response = match querier.raw_query(&raw_query) {
        SystemResult::Ok(ContractResult::Ok(response)) => response,
        SystemResult::Ok(ContractResult::Err(err)) => {
            return Err(StdError::generic_err(format!(
                "stargate contract error: {err}"
            )));
        }
        SystemResult::Err(err) => {
            return Err(StdError::generic_err(format!(
                "stargate system error: {err}"
            )));
        }
    };

    Resp::decode(response.as_slice()).map_err(|e| {
        StdError::parse_err(std::any::type_name::<Resp>(), e.to_string())
    })
}

impl From<cosmwasm_std::Coin> for cosmos::base::v1beta1::Coin {
    fn from(cw_coin: cosmwasm_std::Coin) -> Self {
        cosmos::base::v1beta1::Coin {
            denom: cw_coin.denom,
            amount: cw_coin.amount.to_string(),
        }
    }
}

#[cfg(test)]
mod tests {
    use cosmwasm_std::{
        from_json, Binary, ContractResult, Empty, Querier, QuerierResult,
        QuerierWrapper, QueryRequest, SystemError, SystemResult,
    };
    use prost::Message;

    use super::{query_stargate_proto, NibiruStargateQuery};
    use crate::proto::cosmos;

    struct BankBalanceStargateQuerier {
        expected_path: &'static str,
        response: Binary,
    }

    impl Querier for BankBalanceStargateQuerier {
        fn raw_query(&self, bin_request: &[u8]) -> QuerierResult {
            let request: QueryRequest<Empty> = match from_json(bin_request) {
                Ok(request) => request,
                Err(err) => {
                    return SystemResult::Err(SystemError::InvalidRequest {
                        error: format!("parsing query request: {err}"),
                        request: bin_request.into(),
                    });
                }
            };

            match request {
                QueryRequest::Stargate { path, .. }
                    if path == self.expected_path =>
                {
                    SystemResult::Ok(ContractResult::Ok(self.response.clone()))
                }
                _ => SystemResult::Err(SystemError::UnsupportedRequest {
                    kind: "unexpected query".to_string(),
                }),
            }
        }
    }

    #[test]
    fn query_stargate_proto_decodes_bank_balance_response() {
        let expected = cosmos::bank::v1beta1::QueryBalanceResponse {
            balance: Some(cosmos::base::v1beta1::Coin {
                denom: "unibi".to_string(),
                amount: "123456".to_string(),
            }),
        };
        let querier = BankBalanceStargateQuerier {
            expected_path: "/cosmos.bank.v1beta1.Query/Balance",
            response: Binary::from(expected.encode_to_vec()),
        };
        let wrapper = QuerierWrapper::<Empty>::new(&querier);

        let req = cosmos::bank::v1beta1::QueryBalanceRequest {
            address: "nibi1contract".to_string(),
            denom: "unibi".to_string(),
        };

        let stargate_query = req
            .into_stargate_query()
            .expect("bank balance request should convert to Stargate");
        assert!(matches!(
            stargate_query,
            QueryRequest::Stargate { ref path, .. }
                if path == "/cosmos.bank.v1beta1.Query/Balance"
        ));

        let actual: cosmos::bank::v1beta1::QueryBalanceResponse =
            query_stargate_proto(&wrapper, &req).unwrap();
        assert_eq!(actual, expected);
    }
}

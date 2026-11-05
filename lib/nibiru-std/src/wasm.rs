// crate::wasm.rs

use cosmwasm_std::{QueryRequest, StdResult, WasmQuery};

/// Generic helper for constructing WasmQuery::Smart query requests.
pub fn wasm_query_smart<CosmosMsg>(
    contract: impl Into<String>,
    msg: &impl crate::proto::NibiruProstMsg,
) -> StdResult<QueryRequest<CosmosMsg>> {
    Ok(QueryRequest::Wasm(WasmQuery::Smart {
        contract_addr: contract.into(),
        msg: msg.to_binary(),
    }))
}

/// Generic helper for constructing WasmQuery::Raw query requests.
pub fn wasm_query_raw<CosmosMsg>(
    contract: impl Into<String>,
    key: &str,
) -> StdResult<QueryRequest<CosmosMsg>> {
    Ok(QueryRequest::Wasm(WasmQuery::Raw {
        contract_addr: contract.into(),
        key: key.as_bytes().into(),
    }))
}

#[cfg(test)]
mod tests {
    use prost::Message;

    use super::*;
    use crate::proto::{
        cosmos::{bank, base::v1beta1::Coin},
        nibiru::{self, perp},
        NibiruProstMsg,
    };

    #[test]
    fn test_wasm_query_smart() -> anyhow::Result<()> {
        let proto_msg = nibiru::perp::QueryMarketsRequest { versioned: false };
        let contract: &str = "mock_contract_addr";
        let query_req =
            wasm_query_smart::<cosmwasm_std::Empty>(contract, &proto_msg)?;

        match query_req {
            QueryRequest::Wasm(WasmQuery::Smart {
                contract_addr,
                msg: msg_bz,
            }) => {
                assert_eq!(contract_addr, contract, "{msg_bz:?}");
                assert_eq!(msg_bz, proto_msg.to_binary(), "{msg_bz:?}");
            }
            _ => return Err(anyhow::anyhow!("failed to parse wasm query")),
        }

        Ok(())
    }

    #[test]
    fn proto_msgs_encode() {
        let coin = Coin {
            denom: "nibi".into(),
            amount: "420".into(),
        };
        let mut msg_0a = bank::v1beta1::MsgSend {
            from_address: "from".into(),
            to_address: "to".into(),
            amount: vec![coin.clone()],
        };
        let msg_0b = bank::v1beta1::MsgSend {
            from_address: "from".into(),
            to_address: "to".into(),
            amount: vec![],
        };

        // Different when compared in all forms
        assert_ne!(msg_0a, msg_0b);
        assert_ne!(msg_0a.encode_to_vec(), msg_0b.encode_to_vec());
        assert_ne!(msg_0a.to_binary(), msg_0b.to_binary());

        // Now they should match
        msg_0a.amount = vec![];
        assert_eq!(msg_0a.to_binary(), msg_0b.to_binary());

        let sender = "sender".to_string();
        let pair = "pair".to_string();
        let msg_1a = perp::MsgAddMargin {
            sender: sender.clone(),
            pair: pair.clone(),
            margin: None,
        };
        let mut msg_1b = perp::MsgAddMargin {
            sender,
            pair,
            margin: Some(coin),
        };

        // Different when compared in all forms
        assert_ne!(msg_1a, msg_1b);
        assert_ne!(msg_1a.encode_to_vec(), msg_1b.encode_to_vec());
        assert_ne!(msg_1a.to_binary(), msg_1b.to_binary());

        // Now they should match
        msg_1b.margin = None;
        assert_eq!(msg_1a.to_binary(), msg_1b.to_binary());
    }

    #[test]
    fn test_wasm_query_raw() -> anyhow::Result<()> {
        let query_req =
            wasm_query_raw::<cosmwasm_std::Empty>("contract", "key")?;
        match query_req {
            QueryRequest::Wasm(WasmQuery::Raw { contract_addr, key }) => {
                assert_eq!(contract_addr, "contract");
                assert_eq!(key, "key".as_bytes());
            }
            _ => return Err(anyhow::anyhow!("failed to parse wasm query")),
        }
        Ok(())
    }
}

//! Temporary candidate registry, deployed before the v2.23 upgrade.
//! It holds no balances or funds. The upgrade reads current chain balances.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{
    entry_point, to_json_binary, Addr, Binary, Deps, DepsMut, Env, MessageInfo,
    Order, Response, StdError, StdResult,
};
use cw_storage_plus::{Bound, Item, Map};
use std::collections::BTreeSet;

const ADMIN: Item<Addr> = Item::new("admin");
const ADDRESSES: Map<&str, bool> = Map::new("addresses");
pub const MAX_BATCH: usize = 500;
pub const DEFAULT_LIMIT: u32 = 200;
pub const MAX_LIMIT: u32 = 500;

#[cw_serde]
pub struct InstantiateMsg {
    pub admin: String,
    #[serde(default)]
    pub addresses: Vec<String>,
}

#[cw_serde]
pub enum ExecuteMsg {
    UpdateAddresses {
        #[serde(default)]
        add: Vec<String>,
        #[serde(default)]
        remove: Vec<String>,
    },
}

#[cw_serde]
#[derive(QueryResponses)]
pub enum QueryMsg {
    #[returns(AddressesResponse)]
    Addresses {
        start_after: Option<String>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct AddressesResponse {
    pub addresses: Vec<String>,
    pub next_start_after: Option<String>,
}

fn normalize(address: &str) -> StdResult<String> {
    if address.len() != 42
        || !(address.starts_with("0x") || address.starts_with("0X"))
        || !address.as_bytes()[2..].iter().all(u8::is_ascii_hexdigit)
    {
        return Err(StdError::generic_err(
            "expected a 20-byte hexadecimal EVM address",
        ));
    }
    Ok(address.to_ascii_lowercase())
}

fn validate_batch(addresses: Vec<String>) -> StdResult<BTreeSet<String>> {
    if addresses.len() > MAX_BATCH {
        return Err(StdError::generic_err("address batch exceeds 500 entries"));
    }
    addresses.iter().map(|a| normalize(a)).collect()
}

fn no_funds(info: &MessageInfo) -> StdResult<()> {
    if !info.funds.is_empty() {
        return Err(StdError::generic_err("holder set does not accept funds"));
    }
    Ok(())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: InstantiateMsg,
) -> StdResult<Response> {
    no_funds(&info)?;
    let admin = deps.api.addr_validate(&msg.admin)?;
    let addresses = validate_batch(msg.addresses)?;
    ADMIN.save(deps.storage, &admin)?;
    for address in addresses {
        ADDRESSES.save(deps.storage, &address, &true)?;
    }
    Ok(Response::new().add_attribute("action", "instantiate_holder_set"))
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    if info.sender != ADMIN.load(deps.storage)? {
        return Err(StdError::generic_err("unauthorized"));
    }
    no_funds(&info)?;
    match msg {
        ExecuteMsg::UpdateAddresses { add, remove } => {
            // Validate the whole message before changing any storage.
            let add = validate_batch(add)?;
            let remove = validate_batch(remove)?;
            if !add.is_disjoint(&remove) {
                return Err(StdError::generic_err(
                    "address occurs in both add and remove",
                ));
            }
            for address in &remove {
                ADDRESSES.remove(deps.storage, address);
            }
            for address in &add {
                ADDRESSES.save(deps.storage, address, &true)?;
            }
            Ok(Response::new()
                .add_attribute("action", "update_addresses")
                .add_attribute("added", add.len().to_string())
                .add_attribute("removed", remove.len().to_string()))
        }
    }
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Addresses { start_after, limit } => {
            let limit = limit.unwrap_or(DEFAULT_LIMIT);
            if limit == 0 || limit > MAX_LIMIT {
                return Err(StdError::generic_err(
                    "limit must be between 1 and 500",
                ));
            }
            let start_after =
                start_after.as_deref().map(normalize).transpose()?;
            // Read one extra key to distinguish a full final page from more data.
            let mut addresses = ADDRESSES
                .keys(
                    deps.storage,
                    start_after.as_deref().map(Bound::exclusive),
                    None,
                    Order::Ascending,
                )
                .take(limit as usize + 1)
                .collect::<StdResult<Vec<_>>>()?;
            let has_more = addresses.len() > limit as usize;
            addresses.truncate(limit as usize);
            let next_start_after = if has_more {
                addresses.last().cloned()
            } else {
                None
            };
            to_json_binary(&AddressesResponse {
                addresses,
                next_start_after,
            })
        }
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use cosmwasm_std::{
        from_json,
        testing::{mock_dependencies, mock_env, mock_info},
    };

    fn address(n: u64) -> String {
        format!("0x{n:040x}")
    }

    #[test]
    fn updates_are_authorized_normalized_and_validated_before_writes() {
        let mut deps = mock_dependencies();
        instantiate(
            deps.as_mut(),
            mock_env(),
            mock_info("creator", &[]),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: vec![address(10).to_uppercase(), address(10)],
            },
        )
        .unwrap();
        let msg = ExecuteMsg::UpdateAddresses {
            add: vec![address(11)],
            remove: vec![address(10)],
        };
        assert!(execute(
            deps.as_mut(),
            mock_env(),
            mock_info("outsider", &[]),
            msg.clone()
        )
        .is_err());
        execute(deps.as_mut(), mock_env(), mock_info("admin", &[]), msg)
            .unwrap();
        let bad = ExecuteMsg::UpdateAddresses {
            add: vec![address(12), "0x123".into()],
            remove: vec![address(11)],
        };
        assert!(execute(
            deps.as_mut(),
            mock_env(),
            mock_info("admin", &[]),
            bad
        )
        .is_err());
        let page: AddressesResponse = from_json(
            query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::Addresses {
                    start_after: None,
                    limit: None,
                },
            )
            .unwrap(),
        )
        .unwrap();
        assert_eq!(page.addresses, vec![address(11)]);
        assert_eq!(page.next_start_after, None);
    }

    #[test]
    fn pagination_is_exclusive_complete_and_reports_exact_final_pages() {
        let mut deps = mock_dependencies();
        instantiate(
            deps.as_mut(),
            mock_env(),
            mock_info("creator", &[]),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: (0..6).rev().map(address).collect(),
            },
        )
        .unwrap();
        let mut cursor = None;
        let mut result = vec![];
        loop {
            let page: AddressesResponse = from_json(
                query(
                    deps.as_ref(),
                    mock_env(),
                    QueryMsg::Addresses {
                        start_after: cursor,
                        limit: Some(2),
                    },
                )
                .unwrap(),
            )
            .unwrap();
            result.extend(page.addresses);
            cursor = page.next_start_after;
            if cursor.is_none() {
                break;
            }
        }
        assert_eq!(result, (0..6).map(address).collect::<Vec<_>>());
        for limit in [0, 501] {
            assert!(query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::Addresses {
                    start_after: None,
                    limit: Some(limit)
                }
            )
            .is_err());
        }
        assert!(query(
            deps.as_ref(),
            mock_env(),
            QueryMsg::Addresses {
                start_after: Some("bad".into()),
                limit: None
            }
        )
        .is_err());
    }
}

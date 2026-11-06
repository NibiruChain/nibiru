//! Temporary candidate registry, deployed before the v2.23 upgrade.
//! It holds no balances or funds. The upgrade reads current chain balances.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{
    entry_point, to_json_binary, Addr, Binary, Deps, DepsMut, Env, MessageInfo,
    Order, Response, StdError, StdResult,
};
use cw_storage_plus::{Bound, Item, Map};
use nibiru_std::address::UserAddr;
use std::collections::BTreeSet;

const ADMIN: Item<Addr> = Item::new("admin");
const HOLDING_ADDRESS: Item<UserAddr> = Item::new("holding_address");
const MAINNET: &str = "cataclysm-1";
const TESTNET: &str = "nibiru-testnet-2";
const LOCALNET: &str = "nibiru-localnet-0";
const SAFE: &str = "0x22CBd7CbF3b33681abB3Ced4D64d71acB9a9dCd2";
const OP_B: &str = "0x447fC0471f5d8150A790e19000930108aCCe0BC7";
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
    SetHoldingAddress {
        holding_address: UserAddr,
    },
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
    #[returns(ConfigResponse)]
    Config {},
    #[returns(AddressesResponse)]
    Addresses {
        start_after: Option<String>,
        limit: Option<u32>,
    },
}

#[cw_serde]
pub struct ConfigResponse {
    pub chain_id: String,
    pub holding_address: UserAddr,
    pub admin: Addr,
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
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> StdResult<Response> {
    if info.sender != ADMIN.load(deps.storage)? {
        return Err(StdError::generic_err("unauthorized"));
    }
    no_funds(&info)?;
    match msg {
        ExecuteMsg::SetHoldingAddress { holding_address } => {
            match env.block.chain_id.as_str() {
                MAINNET => {
                    return Err(StdError::generic_err(
                        "mainnet holding address is immutable",
                    ))
                }
                TESTNET | LOCALNET => {}
                _ => return Err(StdError::generic_err("unsupported chain ID")),
            }
            if holding_address != parse_target(SAFE)?
                && holding_address != parse_target(OP_B)?
            {
                return Err(StdError::generic_err(
                    "holding address must be the approved Safe or op-b",
                ));
            }
            HOLDING_ADDRESS.save(deps.storage, &holding_address)?;
            Ok(Response::new()
                .add_attribute("action", "set_holding_address")
                .add_attribute("holding_address", holding_address.to_hex()))
        }
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

fn parse_target(address: &str) -> StdResult<UserAddr> {
    address
        .parse()
        .map_err(|err: nibiru_std::errors::NibiruError| {
            StdError::generic_err(err.to_string())
        })
}

fn query_config(deps: Deps, env: Env) -> StdResult<ConfigResponse> {
    let holding_address = match env.block.chain_id.as_str() {
        MAINNET => parse_target(SAFE)?,
        TESTNET | LOCALNET => HOLDING_ADDRESS
            .may_load(deps.storage)?
            .unwrap_or(parse_target(OP_B)?),
        _ => return Err(StdError::generic_err("unsupported chain ID")),
    };
    Ok(ConfigResponse {
        chain_id: env.block.chain_id,
        holding_address,
        admin: ADMIN.load(deps.storage)?,
    })
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Config {} => to_json_binary(&query_config(deps, env)?),
        QueryMsg::Addresses { start_after, limit } => {
            let limit = limit.unwrap_or(DEFAULT_LIMIT);
            if limit == 0 || limit > MAX_LIMIT {
                return Err(StdError::generic_err(
                    "limit must be between 1 and 500",
                ));
            }
            // Pagination cursors are opaque stored keys, not candidates to
            // validate. A malformed key returned at a page boundary must still
            // let the reader resume after it and reach later valid holders.
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
        let page: AddressesResponse = from_json(
            query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::Addresses {
                    start_after: Some("bad".into()),
                    limit: None,
                },
            )
            .unwrap(),
        )
        .unwrap();
        assert!(page.addresses.is_empty());
        assert_eq!(page.next_start_after, None);
    }

    #[test]
    fn opaque_cursor_resumes_after_a_malformed_boundary_key() {
        let mut deps = mock_dependencies();
        instantiate(
            deps.as_mut(),
            mock_env(),
            mock_info("creator", &[]),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: (1..=201).map(address).collect(),
            },
        )
        .unwrap();
        // Admin updates still reject malformed candidates. Inject storage only
        // to prove defensive pagination can cross a malformed returned key.
        let bad = format!("{}z", address(199));
        ADDRESSES.save(deps.as_mut().storage, &bad, &true).unwrap();
        let first: AddressesResponse = from_json(
            query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::Addresses {
                    start_after: None,
                    limit: Some(200),
                },
            )
            .unwrap(),
        )
        .unwrap();
        assert_eq!(first.addresses.len(), 200);
        assert_eq!(first.addresses.last(), Some(&bad));
        assert_eq!(first.next_start_after, Some(bad.clone()));
        let second: AddressesResponse = from_json(
            query(
                deps.as_ref(),
                mock_env(),
                QueryMsg::Addresses {
                    start_after: first.next_start_after,
                    limit: Some(200),
                },
            )
            .unwrap(),
        )
        .unwrap();
        assert_eq!(second.addresses, vec![address(200), address(201)]);
        assert_eq!(second.next_start_after, None);
    }

    #[test]
    fn config_defaults_use_exact_chain_policies() {
        for (chain_id, expected) in
            [(MAINNET, SAFE), (TESTNET, OP_B), (LOCALNET, OP_B)]
        {
            let mut deps = mock_dependencies();
            let mut env = mock_env();
            env.block.chain_id = chain_id.into();
            instantiate(
                deps.as_mut(),
                env.clone(),
                mock_info("creator", &[]),
                InstantiateMsg {
                    admin: "admin".into(),
                    addresses: vec![],
                },
            )
            .unwrap();
            let encoded =
                query(deps.as_ref(), env, QueryMsg::Config {}).unwrap();
            let config: ConfigResponse = from_json(encoded.clone()).unwrap();
            assert_eq!(config.chain_id, chain_id);
            assert_eq!(config.admin, Addr::unchecked("admin"));
            assert_eq!(config.holding_address, parse_target(expected).unwrap());
            assert!(String::from_utf8(encoded.to_vec())
                .unwrap()
                .contains(&parse_target(expected).unwrap().to_hex()));
        }
    }

    #[test]
    fn test_network_admin_can_select_only_the_two_approved_targets() {
        for chain_id in [TESTNET, LOCALNET] {
            let mut deps = mock_dependencies();
            let mut env = mock_env();
            env.block.chain_id = chain_id.into();
            instantiate(
                deps.as_mut(),
                env.clone(),
                mock_info("creator", &[]),
                InstantiateMsg {
                    admin: "admin".into(),
                    addresses: vec![],
                },
            )
            .unwrap();
            let safe = parse_target(SAFE).unwrap();
            let set_safe = ExecuteMsg::SetHoldingAddress {
                holding_address: safe,
            };
            assert!(execute(
                deps.as_mut(),
                env.clone(),
                mock_info("outsider", &[]),
                set_safe
            )
            .is_err());
            assert_eq!(
                query_config(deps.as_ref(), env.clone())
                    .unwrap()
                    .holding_address,
                parse_target(OP_B).unwrap()
            );
            // UserAddr accepts equivalent bech32 identities and emits EIP-55.
            let msg: ExecuteMsg = from_json(
                format!(
                    r#"{{"set_holding_address":{{"holding_address":"{}"}}}}"#,
                    safe.to_bech32_addr()
                )
                .as_bytes(),
            )
            .unwrap();
            execute(deps.as_mut(), env.clone(), mock_info("admin", &[]), msg)
                .unwrap();
            assert_eq!(
                query_config(deps.as_ref(), env.clone())
                    .unwrap()
                    .holding_address,
                safe
            );
            for unapproved in [address(42), address(0)] {
                assert!(execute(
                    deps.as_mut(),
                    env.clone(),
                    mock_info("admin", &[]),
                    ExecuteMsg::SetHoldingAddress {
                        holding_address: unapproved.parse().unwrap()
                    }
                )
                .is_err());
                assert_eq!(
                    query_config(deps.as_ref(), env.clone())
                        .unwrap()
                        .holding_address,
                    safe
                );
            }
            execute(
                deps.as_mut(),
                env.clone(),
                mock_info("admin", &[]),
                ExecuteMsg::SetHoldingAddress {
                    holding_address: parse_target(OP_B).unwrap(),
                },
            )
            .unwrap();
            assert_eq!(
                query_config(deps.as_ref(), env).unwrap().holding_address,
                parse_target(OP_B).unwrap()
            );
        }
    }

    #[test]
    fn mainnet_destination_is_immutable_even_with_test_override_storage() {
        let mut deps = mock_dependencies();
        let mut env = mock_env();
        env.block.chain_id = MAINNET.into();
        instantiate(
            deps.as_mut(),
            env.clone(),
            mock_info("creator", &[]),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: vec![],
            },
        )
        .unwrap();
        HOLDING_ADDRESS
            .save(deps.as_mut().storage, &parse_target(OP_B).unwrap())
            .unwrap();
        for target in [SAFE, OP_B] {
            let error = execute(
                deps.as_mut(),
                env.clone(),
                mock_info("admin", &[]),
                ExecuteMsg::SetHoldingAddress {
                    holding_address: parse_target(target).unwrap(),
                },
            )
            .unwrap_err();
            assert!(error.to_string().contains("immutable"));
            assert_eq!(
                query_config(deps.as_ref(), env.clone())
                    .unwrap()
                    .holding_address,
                parse_target(SAFE).unwrap()
            );
        }
        execute(
            deps.as_mut(),
            env.clone(),
            mock_info("admin", &[]),
            ExecuteMsg::UpdateAddresses {
                add: vec![address(1)],
                remove: vec![],
            },
        )
        .unwrap();
        let holders: AddressesResponse = from_json(
            query(
                deps.as_ref(),
                env,
                QueryMsg::Addresses {
                    start_after: None,
                    limit: None,
                },
            )
            .unwrap(),
        )
        .unwrap();
        assert_eq!(holders.addresses, vec![address(1)]);
    }

    #[test]
    fn unsupported_networks_reject_config_without_changing_the_holder_api() {
        for chain_id in [
            "unknown",
            "localnet",
            "nibiru-testnet-3",
            "cataclysm-1-extra",
        ] {
            let mut deps = mock_dependencies();
            let mut env = mock_env();
            env.block.chain_id = chain_id.into();
            instantiate(
                deps.as_mut(),
                env.clone(),
                mock_info("creator", &[]),
                InstantiateMsg {
                    admin: "admin".into(),
                    addresses: vec![address(1)],
                },
            )
            .unwrap();
            assert!(
                query(deps.as_ref(), env.clone(), QueryMsg::Config {}).is_err()
            );
            assert!(execute(
                deps.as_mut(),
                env.clone(),
                mock_info("admin", &[]),
                ExecuteMsg::SetHoldingAddress {
                    holding_address: parse_target(OP_B).unwrap()
                }
            )
            .is_err());
            assert_eq!(
                HOLDING_ADDRESS.may_load(deps.as_ref().storage).unwrap(),
                None
            );
            let holders: AddressesResponse = from_json(
                query(
                    deps.as_ref(),
                    env,
                    QueryMsg::Addresses {
                        start_after: None,
                        limit: None,
                    },
                )
                .unwrap(),
            )
            .unwrap();
            assert_eq!(holders.addresses, vec![address(1)]);
        }
    }

    #[test]
    fn every_write_rejects_attached_funds_before_state_changes() {
        use cosmwasm_std::coins;
        let mut deps = mock_dependencies();
        let mut env = mock_env();
        env.block.chain_id = TESTNET.into();
        assert!(instantiate(
            deps.as_mut(),
            env.clone(),
            mock_info("creator", &coins(1, "unibi")),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: vec![]
            }
        )
        .is_err());
        assert_eq!(ADMIN.may_load(deps.as_ref().storage).unwrap(), None);
        instantiate(
            deps.as_mut(),
            env.clone(),
            mock_info("creator", &[]),
            InstantiateMsg {
                admin: "admin".into(),
                addresses: vec![],
            },
        )
        .unwrap();
        for msg in [
            ExecuteMsg::SetHoldingAddress {
                holding_address: parse_target(SAFE).unwrap(),
            },
            ExecuteMsg::UpdateAddresses {
                add: vec![address(1)],
                remove: vec![],
            },
        ] {
            assert!(execute(
                deps.as_mut(),
                env.clone(),
                mock_info("admin", &coins(1, "unibi")),
                msg
            )
            .is_err());
        }
        assert_eq!(
            HOLDING_ADDRESS.may_load(deps.as_ref().storage).unwrap(),
            None
        );
        assert!(!ADDRESSES.has(deps.as_ref().storage, &address(1)));
    }
}

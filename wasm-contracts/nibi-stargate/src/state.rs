use cosmwasm_std::Uint256;
use cw_storage_plus::Map;

pub const TOKEN_SUPPLY: Map<&str, Uint256> = Map::new("token_supply");

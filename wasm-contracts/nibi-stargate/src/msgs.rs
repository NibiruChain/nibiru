use cosmwasm_schema::cw_serde;
use cosmwasm_std::Coin;

#[cw_serde]
pub struct InstantiateMsg {}

#[cw_serde]
pub enum ExecuteMsg {
    // For x/tokenfactory
    CreateDenom { subdenom: String },
    Mint { coin: Coin, mint_to: String },
    Burn { coin: Coin, burn_from: String },
    ChangeAdmin { denom: String, new_admin: String },
}

#[derive(cosmwasm_schema::QueryResponses)]
#[cw_serde]
pub enum QueryMsg {}

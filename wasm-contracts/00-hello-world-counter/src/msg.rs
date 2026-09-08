use cosmwasm_schema::cw_serde;

#[cw_serde]
pub enum ExecuteMsg {
    Increment {},         // Increase count by 1
    Reset { count: i64 }, // Reset to any i64 value
}

#[cw_serde]
#[derive(cosmwasm_schema::QueryResponses)]
pub enum QueryMsg {
    // Count returns the JSON-encoded state
    #[returns(crate::state::State)]
    Count {},
}

#[cw_serde]
pub struct InstantiateMsg {
    pub count: i64,
}

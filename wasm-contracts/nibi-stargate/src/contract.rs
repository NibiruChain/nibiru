use crate::error::ContractError;
use crate::msgs::{ExecuteMsg, InstantiateMsg};
use crate::state::TOKEN_SUPPLY;
use cosmwasm_std::{
    entry_point, CosmosMsg, DepsMut, Env, MessageInfo, Response, StdError,
    Uint256,
};
use nibiru_std::proto::cosmos::{self, base};
use nibiru_std::proto::{nibiru, NibiruStargateMsg};

#[entry_point]
pub fn instantiate(
    _deps: DepsMut,
    _env: Env,
    _: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    Ok(Response::new())
}

#[entry_point]
pub fn execute(
    _deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    let contract_addr: String = env.contract.address.into();
    match msg {
        ExecuteMsg::CreateDenom { subdenom } => {
            let cosmos_msg: CosmosMsg = nibiru::tokenfactory::MsgCreateDenom {
                sender: contract_addr,
                subdenom,
            }
            .into_stargate_msg();

            Ok(Response::new()
                // .add_event()
                .add_message(cosmos_msg))
        }

        ExecuteMsg::Mint { coin, mint_to } => {
            let cosmos_msg: CosmosMsg = nibiru::tokenfactory::MsgMint {
                sender: contract_addr,
                // TODO feat: cosmwasm-std Coin should implement into()
                // base::v1beta1::Coin.
                coin: Some(cosmos::base::v1beta1::Coin {
                    denom: coin.denom.clone(),
                    amount: coin.amount.into(),
                }),
                mint_to,
            }
            .into_stargate_msg();

            let denom_parts: Vec<&str> = coin.denom.split('/').collect();
            if denom_parts.len() != 3 {
                return Err(StdError::generic_err(
                    "invalid denom input".to_string(),
                )
                .into());
            }

            let subdenom = denom_parts[2];
            let supply_key = subdenom;
            let token_supply =
                TOKEN_SUPPLY.may_load(_deps.storage, supply_key)?;
            match token_supply {
                Some(supply) => {
                    let new_supply = supply + Uint256::from(coin.amount);
                    TOKEN_SUPPLY.save(_deps.storage, supply_key, &new_supply)
                }?,
                None => TOKEN_SUPPLY.save(
                    _deps.storage,
                    supply_key,
                    &Uint256::from(coin.amount),
                )?,
            }

            Ok(Response::new()
                // .add_event()
                .add_message(cosmos_msg))
        }

        ExecuteMsg::Burn { coin, burn_from } => {
            let cosmos_msg: CosmosMsg = nibiru::tokenfactory::MsgBurn {
                sender: contract_addr,
                // TODO cosmwasm-std Coin should implement into()
                // base::v1beta1::Coin.
                coin: Some(base::v1beta1::Coin {
                    denom: coin.denom.clone(),
                    amount: coin.amount.into(),
                }),
                burn_from,
            }
            .into_stargate_msg();
            Ok(Response::new()
                // .add_event()
                .add_message(cosmos_msg))
        }

        ExecuteMsg::ChangeAdmin { denom, new_admin } => {
            let cosmos_msg: CosmosMsg = nibiru::tokenfactory::MsgChangeAdmin {
                sender: contract_addr,
                denom: denom.to_string(),
                new_admin: new_admin.to_string(),
            }
            .into_stargate_msg();
            Ok(Response::new()
                // .add_event()
                .add_message(cosmos_msg))
        }
    }
}

// TODO
// #[entry_point]
// pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
//     todo!()
//     // match msg {}
// }

#[cfg(test)]
mod tests {

    use crate::contract::instantiate;
    use crate::error::ContractError;
    use crate::msgs::InstantiateMsg;
    use cosmwasm_std::testing::{mock_env, mock_info};

    use cosmwasm_std as cw;
    use cosmwasm_std::DepsMut;
    use cw::testing::mock_dependencies;

    fn init(deps: DepsMut) -> Result<cw::Response, ContractError> {
        instantiate(deps, mock_env(), mock_info("none", &[]), InstantiateMsg {})
    }

    #[test]
    fn init_runs() -> Result<(), ContractError> {
        let mut deps = mock_dependencies();
        let _env = mock_env();
        let _ = init(deps.as_mut())?;
        Ok(())
    }
}

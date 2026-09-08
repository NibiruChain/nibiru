use crate::error::ContractError;
use crate::events::{
    event_coins_locked, event_funds_withdrawn, event_unlock_initiated,
};
use crate::msgs::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{
    locks, Lock, LockState, LOCKS_ID, NOT_UNLOCKING_BLOCK_IDENTIFIER,
};
use cosmwasm_std::{
    to_json_binary, BankMsg, Binary, CosmosMsg, Deps, DepsMut, Env, Event,
    MessageInfo, Order, Response, StdResult,
};
use cw_storage_plus::Bound;

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _: MessageInfo,
    _msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    LOCKS_ID.save(deps.storage, &0).unwrap();

    Ok(Response::new())
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::Lock { blocks } => execute_lock(deps, env, info, blocks),

        ExecuteMsg::InitiateUnlock { id } => {
            execute_initiate_unlock(deps, env, info, id)
        }

        ExecuteMsg::WithdrawFunds { id } => {
            execute_withdraw_funds(deps, env, info, id)
        }
    }
}

pub(crate) fn execute_withdraw_funds(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    id: u64,
) -> Result<Response, ContractError> {
    let locks = locks();

    let mut tx_msgs: Vec<CosmosMsg> = Vec::new();

    // we update the lock to mark funds have been withdrawn
    let lock =
        locks.update(deps.storage, id, |lock| -> Result<_, ContractError> {
            let mut lock = lock.ok_or(ContractError::NotFound(id))?;
            match lock.state(env.block.height) {
                LockState::Matured => {
                    tx_msgs.push(
                        BankMsg::Send {
                            to_address: lock.owner.to_string(),
                            amount: vec![lock.coin.clone()],
                        }
                        .into(),
                    );
                    lock.funds_withdrawn = true;
                    Ok(lock)
                }
                LockState::FundedPreUnlock | LockState::Unlocking => {
                    Err(ContractError::NotMatured(id))
                }
                LockState::Withdrawn => {
                    Err(ContractError::FundsAlreadyWithdrawn(id))
                }
            }
        })?;

    Ok(Response::new()
        .add_event(event_funds_withdrawn(id, &lock.coin))
        .add_messages(tx_msgs))
}

pub(crate) fn execute_initiate_unlock(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    id: u64,
) -> Result<Response, ContractError> {
    let locks = locks();

    // initiate unlock
    let lock =
        locks.update(deps.storage, id, |lock| -> Result<_, ContractError> {
            let mut lock = lock.ok_or(ContractError::NotFound(id))?;

            match lock.state(env.block.height) {
                LockState::FundedPreUnlock => {
                    lock.end_block = env.block.height + lock.duration_blocks;
                    Ok(lock)
                }
                LockState::Unlocking | LockState::Matured => {
                    Err(ContractError::AlreadyUnlocking(id))
                }
                LockState::Withdrawn => {
                    Err(ContractError::FundsAlreadyWithdrawn(id))
                }
            }
        })?;

    // emit unlock initiation event
    Ok(Response::new().add_event(event_unlock_initiated(
        id,
        &lock.coin,
        lock.end_block,
    )))
}

pub(crate) fn execute_lock(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    blocks: u64,
) -> Result<Response, ContractError> {
    if blocks == 0 {
        return Err(ContractError::InvalidLockDuration);
    }

    if info.funds.is_empty() {
        return Err(ContractError::InvalidCoins("no funds".to_string()));
    }

    // create a lock for each coin sent
    let locks = locks();
    let mut events: Vec<Event> = Vec::with_capacity(info.funds.len());
    for coin in info.funds {
        if coin.amount.is_zero() {
            return Err(ContractError::InvalidCoins(format!(
                "zero coins in denom: {}",
                coin.denom
            )));
        }

        let id = LOCKS_ID
            .update(deps.storage, |id| -> StdResult<_> { Ok(id + 1) })
            .expect("must never fail");

        locks
            .save(
                deps.storage,
                id,
                &Lock {
                    id,
                    coin: coin.clone(),
                    owner: info.sender.to_string(),
                    duration_blocks: blocks,
                    start_block: env.block.height,
                    end_block: NOT_UNLOCKING_BLOCK_IDENTIFIER,
                    funds_withdrawn: false,
                },
            )
            .expect("must never fail");

        events.push(event_coins_locked(id, &coin))
    }

    Ok(Response::new().add_events(events))
}

#[cfg_attr(not(feature = "library"), cosmwasm_std::entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::LocksByDenomAndAddressUnlockingAfter {
            denom,
            unlocking_after,
            address,
        } => Ok(to_json_binary(
            &locks()
                .idx
                .addr_denom_end
                .sub_prefix((address, denom))
                .range(
                    deps.storage,
                    Some(Bound::inclusive((unlocking_after, 0))),
                    None,
                    Order::Ascending,
                )
                .map(|lock| -> Lock { lock.unwrap().1 })
                .filter(|lock| -> bool {
                    if env.block.height + lock.duration_blocks <= unlocking_after
                    {
                        return false;
                    }
                    true
                })
                .collect::<Vec<Lock>>(),
        )?),

        QueryMsg::LocksByDenomUnlockingAfter {
            denom,
            unlocking_after,
        } => Ok(to_json_binary(
            &locks()
                .idx
                .denom_end
                .sub_prefix(denom)
                .range(
                    deps.storage,
                    Some(Bound::inclusive((unlocking_after, 0))),
                    None,
                    Order::Ascending,
                )
                .map(|lock| -> Lock { lock.unwrap().1 })
                .filter(|lock| -> bool {
                    // TODO: make this more efficient by indexing this in state
                    if env.block.height + lock.duration_blocks <= unlocking_after
                    {
                        return false;
                    }
                    true
                })
                .collect::<Vec<Lock>>(),
        )?),

        QueryMsg::LocksByDenomBetween {
            denom,
            locked_before,
            unlocking_after,
        } => Ok(to_json_binary(
            &locks()
                .idx
                .denom_start
                .sub_prefix(denom)
                .range(
                    deps.storage,
                    None,
                    Some(Bound::exclusive((locked_before, u64::MAX))),
                    Order::Ascending,
                )
                .map(|lock| -> Lock { lock.unwrap().1 })
                .filter(|lock| -> bool {
                    env.block.height + lock.duration_blocks > unlocking_after
                })
                .collect::<Vec<Lock>>(),
        )?),

        QueryMsg::LocksByDenomAndAddressBetween {
            denom,
            address,
            locked_before,
            unlocking_after,
        } => Ok(to_json_binary(
            &locks()
                .idx
                .addr_denom_start
                .sub_prefix((address, denom))
                .range(
                    deps.storage,
                    None,
                    Some(Bound::exclusive((locked_before, u64::MAX))),
                    Order::Ascending,
                )
                .map(|lock| -> Lock { lock.unwrap().1 })
                .filter(|lock| -> bool {
                    lock.duration_blocks + env.block.height > unlocking_after
                })
                .collect::<Vec<Lock>>(),
        )?),
    }
}

#[cfg(test)]
mod tests {

    use crate::contract::{execute_lock, instantiate, query};
    use crate::msgs::{InstantiateMsg, QueryMsg};
    use crate::state::Lock;
    use cosmwasm_std::testing::{mock_dependencies, mock_env, mock_info};

    use cosmwasm_std::{from_json, Coin, DepsMut, Env};

    /// A 'TestLock' is struct representating an "owner" locking "coins" for
    /// some "duration".
    struct TestLock {
        owner: String,
        duration: u64,
        coins: Vec<Coin>,
    }

    fn init(deps: DepsMut) {
        instantiate(deps, mock_env(), mock_info("none", &[]), InstantiateMsg {})
            .unwrap();
    }

    /// Calls execute_lock on the given test lock.
    fn create_lock(deps: DepsMut, env: &Env, lock: &TestLock) {
        execute_lock(
            deps,
            env.clone(),
            mock_info(lock.owner.to_string().as_str(), lock.coins.as_slice()),
            lock.duration,
        )
        .unwrap();
    }

    /// msg: The query message, which is assumped to return a "Vec<Lock>" after
    ///     unwrapping the binary into a concrete type.
    /// coins (Vec<Coin>): the expected coins on the lock returned by the query.
    fn test_query_locks_for_coins(
        msg: &QueryMsg,
        coins: &[Coin],
        deps: DepsMut,
        env: Env,
    ) {
        assert_eq!(
            coins,
            from_json::<Vec<Lock>>(
                &query(deps.as_ref(), env, msg.clone()).unwrap()
            )
            .unwrap()
            .iter()
            .map(|lock| -> Coin { lock.coin.clone() })
            .collect::<Vec<Coin>>()
        );
    }

    #[test]
    fn queries() {
        let mut deps = mock_dependencies();
        let env = mock_env();

        init(deps.as_mut());

        let lock_1 = TestLock {
            owner: String::from("alice"),
            duration: 100,
            coins: vec![Coin::new(100u128, "ATOM"), Coin::new(300u128, "LUNA")],
        };

        let lock_2 = TestLock {
            owner: String::from("bob"),
            duration: 50,
            coins: vec![Coin::new(200u128, "ATOM"), Coin::new(700u128, "NIBI")],
        };
        create_lock(deps.as_mut(), &env, &lock_1);
        create_lock(deps.as_mut(), &env, &lock_2);

        // QueryMsg::LocksByDenomUnlockingAfter should should the full lock.coin
        // amounts if unlocking_after is zero.
        // let unlocking_after = 0;
        struct CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg,
            coins: Vec<Coin>,
        }
        let mut cases: Vec<CaseLocksByDenomUnlockingAfter> = Vec::new();
        let unlocking_after = 0;
        let denom = "ATOM";
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after,
            },
            coins: vec![Coin::new(100u128, denom), Coin::new(200u128, denom)],
        });
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomAndAddressUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after: 0,
                address: lock_1.owner.clone(),
            },
            coins: vec![Coin::new(100u128, denom)],
        });
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomAndAddressUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after: 0,
                address: lock_2.owner.clone(),
            },
            coins: vec![Coin::new(200u128, denom)],
        });

        let denom = "LUNA";
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after,
            },
            coins: vec![Coin::new(300u128, denom)],
        });

        let denom = "NIBI";
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after,
            },
            coins: vec![Coin::new(700u128, denom)],
        });

        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomAndAddressUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after,
                address: lock_1.owner,
            },
            coins: vec![],
        });
        cases.push(CaseLocksByDenomUnlockingAfter {
            msg: QueryMsg::LocksByDenomAndAddressUnlockingAfter {
                denom: denom.to_string(),
                unlocking_after,
                address: lock_2.owner,
            },
            coins: vec![Coin::new(700u128, denom)],
        });

        for case in &cases {
            test_query_locks_for_coins(
                &case.msg,
                &case.coins,
                deps.as_mut(),
                env.clone(),
            )
        }
    }
}

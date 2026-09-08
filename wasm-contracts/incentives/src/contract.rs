use crate::add_coins;
use crate::error::ContractError;
use crate::events::{new_incentives_program_event, new_program_funding};
use crate::msgs::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{
    funding, EpochInfo, Funding, Program, EPOCH_INFO, FUNDING_ID,
    LAST_EPOCH_PROCESSED, LOCKUP_ADDR, PROGRAMS, PROGRAMS_ID, WITHDRAWALS,
};
use cosmwasm_std::{
    entry_point, to_json_binary, BankMsg, Binary, Coin, Decimal, Deps, DepsMut,
    Env, MessageInfo, Order, Response, StdResult, Uint128,
};
use cw_storage_plus::Bound;
use lockup::msgs::QueryMsg as LockupQueryMsg;
use lockup::state::Lock;

// TODO: test query entry point
#[allow(dead_code)]
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::ProgramFunding { program_id: id } => to_json_binary(
            &funding()
                .idx
                .pay_from_epoch
                .sub_prefix(id)
                .range(deps.storage, None, None, Order::Ascending)
                .map(|res| -> Funding { res.unwrap().1 })
                .collect::<Vec<Funding>>(),
        ),
        QueryMsg::EpochInfo {
            program_id,
            epoch_number,
        } => to_json_binary(
            &EPOCH_INFO.load(deps.storage, (program_id, epoch_number))?,
        ),
    }
}

// TODO: test instantiate entry point
#[allow(dead_code)]
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    PROGRAMS_ID.save(deps.storage, &0).unwrap();
    FUNDING_ID.save(deps.storage, &0).unwrap();
    LOCKUP_ADDR
        .save(deps.storage, &msg.lockup_contract_address)
        .unwrap(); // TODO(mercilex): maybe check if addr exist in wasm

    Ok(Response::new())
}

// TODO: test execute entry point
#[allow(dead_code)]
#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    match msg {
        ExecuteMsg::CreateProgram {
            denom,
            epochs,
            epoch_block_duration,
            min_lockup_blocks,
        } => execute_create_program(
            deps,
            env,
            info,
            denom,
            epochs,
            epoch_block_duration,
            min_lockup_blocks,
        ),

        ExecuteMsg::FundProgram { id } => {
            execute_fund_program(deps, env, info, id)
        }

        ExecuteMsg::ProcessEpoch { id } => {
            execute_process_epoch(deps, env, info, id)
        }

        ExecuteMsg::WithdrawRewards { id } => {
            execute_withdraw_rewards(deps, env, info, id)
        }
    }
}

fn execute_fund_program(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    program_id: u64,
) -> Result<Response, ContractError> {
    if info.funds.is_empty() {
        return Err(ContractError::FundsRequired);
    }

    // assert program exists
    let program = PROGRAMS.load(deps.storage, program_id)?;
    // equality because epoch processing can be triggered before funding
    if program.end_block <= env.block.height {
        return Err(ContractError::ProgramFinished(
            program_id,
            program.end_block,
            env.block.height,
        ));
    }

    // prepare response event before
    // so we can avoid to copy funds
    // due to funds being moved
    let response =
        Response::new().add_event(new_program_funding(program_id, &info.funds));

    // update funding associated with the program id for this block
    let pay_from_epoch = calc_epoch_to_pay_from(env.block.height, &program);
    println!("pays from epoch {}", pay_from_epoch);

    for coin in info.funds {
        let funding_id = FUNDING_ID
            .update(deps.storage, |id| -> StdResult<_> { Ok(id + 1) })
            .unwrap();

        funding()
            .save(
                deps.storage,
                funding_id,
                &Funding {
                    id: funding_id,
                    program_id,
                    pay_from_epoch,
                    denom: coin.denom,
                    initial_amount: coin.amount,
                    to_pay_each_epoch: coin.amount
                        / Uint128::from(program.epochs - pay_from_epoch + 1),
                },
            )
            .unwrap();
    }

    Ok(response)
}

fn execute_create_program(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    denom: String,
    epochs: u64,
    epoch_duration_blocks: u64,
    min_lockup_blocks: u64,
) -> Result<Response, ContractError> {
    let id = PROGRAMS_ID
        .update(deps.storage, |id| -> StdResult<u64> { Ok(id + 1) })
        .unwrap();

    let program = Program {
        id,
        epochs,
        epoch_duration: epoch_duration_blocks,
        min_lockup_duration_blocks: min_lockup_blocks,
        lockup_denom: denom,
        start_block: env.block.height,
        end_block: env.block.height + (epochs * epoch_duration_blocks),
    };
    PROGRAMS.save(deps.storage, id, &program)?;

    Ok(Response::new().add_event(new_incentives_program_event(&program)))
}

fn execute_withdraw_rewards(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    program_id: u64,
) -> Result<Response, ContractError> {
    // fetch the program
    let program = PROGRAMS.load(deps.storage, program_id)?;

    // find epochs that need to be paid for this addr
    let epochs_to_pay = EPOCH_INFO
        .prefix(program_id)
        .range(
            deps.storage,
            Some(
                WITHDRAWALS
                    .load(deps.storage, (program_id, info.sender.clone()))
                    .map_or_else(
                        |_| -> Bound<'_, _> { Bound::inclusive(0_u64) },
                        |last_withdrawal| -> Bound<'_, _> {
                            Bound::exclusive(last_withdrawal)
                        },
                    ),
            ),
            None,
            Order::Ascending,
        )
        .map(|epoch| -> EpochInfo { epoch.unwrap().1 })
        .collect::<Vec<EpochInfo>>();

    let mut last_paid_epoch = 0;
    let mut to_distribute: Vec<Coin> = vec![];
    for epoch in epochs_to_pay {
        // query lockup to check if the user has some qualified locks
        let epoch_qualified_locks: Vec<Lock> = deps.querier.query_wasm_smart(
            LOCKUP_ADDR.load(deps.storage).unwrap(),
            &lockup::msgs::QueryMsg::LocksByDenomAndAddressBetween {
                denom: program.lockup_denom.clone(),
                unlocking_after: epoch.for_coins_unlocking_after,
                address: info.sender.to_string(),
                locked_before: epoch.for_coins_locked_before,
            },
        )?;

        println!("{:?}", epoch_qualified_locks.clone());

        // get the total amount of locked coins
        let qualified_locked_amount = epoch_qualified_locks
            .iter()
            .map(|lock| -> Uint128 { lock.coin.amount })
            .sum::<Uint128>();

        // now for each coin to distribute
        // we compute the weight of the
        // of the sender in the qualified locks
        println!(
            "qualified locked amount: {:?}, {:?}",
            qualified_locked_amount, env.block.height
        );
        let user_ownership_ratio =
            Decimal::from_ratio(qualified_locked_amount, epoch.total_locked);
        println!("ownership ratio: {:?}", user_ownership_ratio.to_string());
        for coin in epoch.to_distribute {
            let coin_amount: Uint128 = coin
                .amount
                .checked_multiply_ratio(
                    qualified_locked_amount,
                    epoch.total_locked,
                )
                .unwrap();
            add_coins(
                &mut to_distribute,
                Coin {
                    denom: coin.denom,
                    amount: coin_amount,
                },
            )
        }

        last_paid_epoch = epoch.epoch_identifier;
    }

    if last_paid_epoch == 0 {
        return Err(ContractError::NothingToWithdraw("".to_string()));
    }

    WITHDRAWALS
        .save(
            deps.storage,
            (program_id, info.sender.clone()),
            &last_paid_epoch,
        )
        .unwrap();

    println!("distributing: {:?}", to_distribute);
    let to_distribute: Vec<Coin> = to_distribute
        .into_iter()
        .filter(|coin| !coin.amount.is_zero())
        .collect();

    if to_distribute.is_empty() {
        return Ok(Response::new());
    }
    Ok(Response::new().add_message(BankMsg::Send {
        to_address: info.sender.to_string(),
        amount: to_distribute,
    }))
}

// TODO: test execute_process_epoch
#[allow(dead_code)]
fn execute_process_epoch(
    deps: DepsMut,
    env: Env,
    _info: MessageInfo,
    program_id: u64,
) -> Result<Response, ContractError> {
    let program = PROGRAMS.load(deps.storage, program_id)?;
    let epoch_to_process = LAST_EPOCH_PROCESSED.update(
        deps.storage,
        program_id,
        |epoch| -> StdResult<u64> {
            // if this is the first time processing the epoch
            // then the identifier is 1. Then we increase each time
            // by 1 sequentially.
            Ok(epoch.unwrap_or_default() + 1)
        },
    )?;

    if epoch_to_process > program.epochs {
        return Err(ContractError::EpochOutOfBounds(
            epoch_to_process,
            program_id,
        ));
    }

    // we get at which block the epoch should be processed
    let epoch_process_block =
        program.start_block + (epoch_to_process * program.epoch_duration);
    // epochs can be processed after the end of the epoch process block
    if env.block.height <= epoch_process_block {
        return Err(ContractError::EpochProcessBlock(
            epoch_to_process,
            program_id,
            epoch_process_block,
        ));
    }

    let lockup_qualification_block =
        epoch_process_block + program.min_lockup_duration_blocks;

    // the concept of epoch processing is quite straight forward
    // we just need to know which addresses hadn't withdrawn
    // the lockup denom before a certain block height
    let locks: Vec<Lock> = deps.querier.query_wasm_smart(
        LOCKUP_ADDR.load(deps.storage).unwrap(),
        &LockupQueryMsg::LocksByDenomBetween {
            denom: program.lockup_denom,
            locked_before: epoch_process_block,
            unlocking_after: lockup_qualification_block,
        },
    )?;

    // identify how much is the total locked
    let total_locked =
        locks.iter().map(|lock| lock.coin.amount).sum::<Uint128>();

    // then we identify how many coins we need to pay
    // based on the program funding

    let mut to_distribute = vec![];

    for funds in funding()
        .idx
        .pay_from_epoch
        .sub_prefix(program_id)
        .range(
            deps.storage,
            None, //
            Some(Bound::inclusive((epoch_to_process, u64::MAX))),
            Order::Ascending,
        )
        .map(|funds| -> Coin {
            funds
                .map(|x| -> Coin {
                    Coin {
                        denom: x.1.denom,
                        amount: x.1.to_pay_each_epoch,
                    }
                })
                .unwrap()
        })
    {
        add_coins(&mut to_distribute, funds)
    }

    let epoch_info = EpochInfo {
        epoch_identifier: epoch_to_process,
        for_coins_locked_before: epoch_process_block,
        for_coins_unlocking_after: lockup_qualification_block,
        to_distribute,
        total_locked,
    };

    EPOCH_INFO.save(
        deps.storage,
        (program_id, epoch_to_process),
        &epoch_info,
    )?;

    Ok(Response::new().set_data(to_json_binary(&epoch_info)?)) // TODO: don't like
}

// TODO: this is extremely inefficient and could be solved by simple
// math but rn brain too fried to do divisions
#[allow(dead_code)]
fn calc_epoch_to_pay_from(funding_block: u64, program: &Program) -> u64 {
    for i in 1..=program.epochs {
        let epoch_block = i * program.epoch_duration + program.start_block;
        if epoch_block > funding_block {
            return i;
        }
    }

    todo!("cover")
}

#[cfg(test)]
mod tests {
    #[test]
    fn create_program() {}

    #[test]
    fn it_works() {
        let result = 2 + 2;
        assert_eq!(result, 4);
    }
}

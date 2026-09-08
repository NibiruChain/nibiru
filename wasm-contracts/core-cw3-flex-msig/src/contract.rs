use std::cmp::Ordering;

#[cfg(not(feature = "library"))]
use cosmwasm_std::entry_point;
use cosmwasm_std::{
    to_json_binary, Binary, BlockInfo, CosmosMsg, Deps, DepsMut, Empty, Env,
    MessageInfo, Order, Response, StdResult,
};

use cw2::set_contract_version;

use cw3::{
    Ballot, Proposal, ProposalListResponse, ProposalResponse, Status, Vote,
    VoteInfo, VoteListResponse, VoteResponse, VoterDetail, VoterListResponse,
    VoterResponse, Votes,
};
use cw3_fixed_multisig::state::{next_id, BALLOTS, PROPOSALS};
use cw4::{Cw4Contract, MemberChangedHookMsg, MemberDiff};
use cw_storage_plus::Bound;
use cw_utils::{maybe_addr, Expiration, ThresholdResponse};

use crate::error::ContractError;
use crate::msg::{ExecuteMsg, InstantiateMsg, QueryMsg};
use crate::state::{Config, CONFIG};

// version info for migration info
pub(crate) const CONTRACT_NAME: &str = "crates.io:cw3-flex-multisig";
pub(crate) const CONTRACT_VERSION: &str = env!("CARGO_PKG_VERSION");

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    let group_addr =
        Cw4Contract(deps.api.addr_validate(&msg.group_addr).map_err(|_| {
            ContractError::InvalidGroup {
                addr: msg.group_addr.clone(),
            }
        })?);
    let total_weight = group_addr.total_weight(&deps.querier)?;
    msg.threshold.validate(total_weight)?;

    let proposal_deposit = msg
        .proposal_deposit
        .map(|deposit| deposit.into_checked(deps.as_ref()))
        .transpose()?;

    set_contract_version(deps.storage, CONTRACT_NAME, CONTRACT_VERSION)?;

    let cfg = Config {
        threshold: msg.threshold,
        max_voting_period: msg.max_voting_period,
        group_addr,
        executor: msg.executor,
        proposal_deposit,
    };
    CONFIG.save(deps.storage, &cfg)?;

    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response<Empty>, ContractError> {
    match msg {
        ExecuteMsg::Propose {
            title,
            description,
            msgs,
            latest,
        } => exec_propose(deps, env, info, title, description, msgs, latest),
        ExecuteMsg::Vote { proposal_id, vote } => {
            exec_vote(deps, env, info, proposal_id, vote)
        }
        ExecuteMsg::Execute { proposal_id } => {
            exec_execute(deps, env, info, proposal_id)
        }
        ExecuteMsg::Close { proposal_id } => {
            exec_close(deps, env, info, proposal_id)
        }
        ExecuteMsg::MemberChangedHook(MemberChangedHookMsg { diffs }) => {
            exec_membership_hook(deps, env, info, diffs)
        }
    }
}

pub fn exec_propose(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    title: String,
    description: String,
    msgs: Vec<CosmosMsg>,
    // we ignore earliest
    latest: Option<Expiration>,
) -> Result<Response<Empty>, ContractError> {
    // only members of the multisig can create a proposal
    let cfg = CONFIG.load(deps.storage)?;

    // Check that the native deposit was paid (as needed).
    if let Some(deposit) = cfg.proposal_deposit.as_ref() {
        deposit.check_native_deposit_paid(&info)?;
    }

    // Only members of the multisig can create a proposal
    // Non-voting members are special - they are allowed to create a proposal and
    // therefore "vote", but they aren't allowed to vote otherwise.
    // Such vote is also special, because despite having 0 weight it still counts when
    // counting threshold passing
    let vote_power = cfg
        .group_addr
        .is_member(&deps.querier, &info.sender, None)?
        .ok_or(ContractError::Unauthorized {})?;

    // max expires also used as default
    let max_expires = cfg.max_voting_period.after(&env.block);
    let mut expires = latest.unwrap_or(max_expires);
    let comp = expires.partial_cmp(&max_expires);
    if let Some(Ordering::Greater) = comp {
        expires = max_expires;
    } else if comp.is_none() {
        return Err(ContractError::WrongExpiration {});
    }

    // Take the cw20 token deposit, if required. We do this before
    // creating the proposal struct below so that we can avoid a clone
    // and move the loaded deposit info into it.
    let take_deposit_msg =
        if let Some(deposit_info) = cfg.proposal_deposit.as_ref() {
            deposit_info
                .get_take_deposit_messages(&info.sender, &env.contract.address)?
        } else {
            vec![]
        };

    // create a proposal
    let mut prop = Proposal {
        title,
        description,
        start_height: env.block.height,
        expires,
        msgs,
        status: Status::Open,
        votes: Votes::yes(vote_power),
        threshold: cfg.threshold,
        total_weight: cfg.group_addr.total_weight(&deps.querier)?,
        proposer: info.sender.clone(),
        deposit: cfg.proposal_deposit,
    };
    prop.update_status(&env.block);
    let id = next_id(deps.storage)?;
    PROPOSALS.save(deps.storage, id, &prop)?;

    // add the first yes vote from voter
    let ballot = Ballot {
        weight: vote_power,
        vote: Vote::Yes,
    };
    BALLOTS.save(deps.storage, (id, &info.sender), &ballot)?;

    Ok(Response::new()
        .add_messages(take_deposit_msg)
        .add_attribute("action", "propose")
        .add_attribute("sender", info.sender)
        .add_attribute("proposal_id", id.to_string())
        .add_attribute("status", format!("{:?}", prop.status)))
}

pub fn exec_vote(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    proposal_id: u64,
    vote: Vote,
) -> Result<Response<Empty>, ContractError> {
    // only members of the multisig can vote
    let cfg = CONFIG.load(deps.storage)?;

    // ensure proposal exists and can be voted on
    let mut prop = PROPOSALS.load(deps.storage, proposal_id)?;
    // Allow voting on Passed and Rejected proposals too,
    if ![Status::Open, Status::Passed, Status::Rejected].contains(&prop.status) {
        return Err(ContractError::NotOpen {});
    }
    // if they are not expired
    if prop.expires.is_expired(&env.block) {
        return Err(ContractError::Expired {});
    }

    // Only voting members of the multisig can vote
    // Additional check if weight >= 1
    // use a snapshot of "start of proposal"
    let vote_power = cfg
        .group_addr
        .is_voting_member(&deps.querier, &info.sender, prop.start_height)?
        .ok_or(ContractError::Unauthorized {})?;

    let previous_ballot =
        BALLOTS.may_load(deps.storage, (proposal_id, &info.sender))?;
    if let Some(ballot) = previous_ballot.as_ref() {
        subtract_vote(&mut prop.votes, ballot.vote, ballot.weight);
    }

    let ballot = Ballot {
        weight: vote_power,
        vote,
    };
    BALLOTS.save(deps.storage, (proposal_id, &info.sender), &ballot)?;

    // update vote tally
    prop.votes.add_vote(vote, vote_power);
    prop.status = Status::Open;
    prop.update_status(&env.block);
    PROPOSALS.save(deps.storage, proposal_id, &prop)?;

    Ok(Response::new()
        .add_attribute("action", "vote")
        .add_attribute("sender", info.sender)
        .add_attribute("proposal_id", proposal_id.to_string())
        .add_attribute("status", format!("{:?}", prop.status)))
}

/// Subtracts the `vote` weight from `Votes` if the tally is not already at 0.
fn subtract_vote(votes: &mut Votes, vote: Vote, weight: u64) {
    match vote {
        Vote::Yes => votes.yes = votes.yes.saturating_sub(weight),
        Vote::No => votes.no = votes.no.saturating_sub(weight),
        Vote::Abstain => votes.abstain = votes.abstain.saturating_sub(weight),
        Vote::Veto => votes.veto = votes.veto.saturating_sub(weight),
    }
}

pub fn exec_execute(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    proposal_id: u64,
) -> Result<Response, ContractError> {
    let mut prop = PROPOSALS.load(deps.storage, proposal_id)?;
    // we allow execution even after the proposal "expiration" as long as all vote come in before
    // that point. If it was approved on time, it can be executed any time.
    prop.update_status(&env.block);
    if prop.status != Status::Passed {
        return Err(ContractError::WrongExecuteStatus {});
    }

    let cfg = CONFIG.load(deps.storage)?;
    cfg.authorize(&deps.querier, &info.sender)?;

    // set it to executed
    prop.status = Status::Executed;
    PROPOSALS.save(deps.storage, proposal_id, &prop)?;

    // Unconditionally refund here.
    let response = match prop.deposit {
        Some(deposit) => Response::new()
            .add_message(deposit.get_return_deposit_message(&prop.proposer)?),
        None => Response::new(),
    };

    // dispatch all proposed messages
    Ok(response
        .add_messages(prop.msgs)
        .add_attribute("action", "execute")
        .add_attribute("sender", info.sender)
        .add_attribute("proposal_id", proposal_id.to_string()))
}

pub fn exec_close(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
    proposal_id: u64,
) -> Result<Response<Empty>, ContractError> {
    // anyone can trigger this if the vote passed

    let mut prop = PROPOSALS.load(deps.storage, proposal_id)?;
    if [Status::Executed, Status::Rejected, Status::Passed]
        .contains(&prop.status)
    {
        return Err(ContractError::WrongCloseStatus {});
    }
    // Avoid closing of Passed due to expiration proposals
    if prop.current_status(&env.block) == Status::Passed {
        return Err(ContractError::WrongCloseStatus {});
    }
    if !prop.expires.is_expired(&env.block) {
        return Err(ContractError::NotExpired {});
    }

    // set it to failed
    prop.status = Status::Rejected;
    PROPOSALS.save(deps.storage, proposal_id, &prop)?;

    // Refund the deposit if we have been configured to do so.
    let mut response = Response::new();
    if let Some(deposit) = prop.deposit {
        if deposit.refund_failed_proposals {
            response = response
                .add_message(deposit.get_return_deposit_message(&prop.proposer)?)
        }
    }

    Ok(response
        .add_attribute("action", "close")
        .add_attribute("sender", info.sender)
        .add_attribute("proposal_id", proposal_id.to_string()))
}

pub fn exec_membership_hook(
    deps: DepsMut,
    _env: Env,
    info: MessageInfo,
    _diffs: Vec<MemberDiff>,
) -> Result<Response<Empty>, ContractError> {
    // This is now a no-op
    // But we leave the authorization check as a demo
    let cfg = CONFIG.load(deps.storage)?;
    if info.sender != cfg.group_addr.0 {
        return Err(ContractError::Unauthorized {});
    }

    Ok(Response::default())
}

#[cfg_attr(not(feature = "library"), entry_point)]
pub fn query(deps: Deps, env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Threshold {} => to_json_binary(&query_threshold(deps)?),
        QueryMsg::Proposal { proposal_id } => {
            to_json_binary(&query_proposal(deps, env, proposal_id)?)
        }
        QueryMsg::Vote { proposal_id, voter } => {
            to_json_binary(&query_vote(deps, proposal_id, voter)?)
        }
        QueryMsg::ListProposals { start_after, limit } => {
            to_json_binary(&list_proposals(deps, env, start_after, limit)?)
        }
        QueryMsg::ReverseProposals {
            start_before,
            limit,
        } => to_json_binary(&reverse_proposals(deps, env, start_before, limit)?),
        QueryMsg::ListVotes {
            proposal_id,
            start_after,
            limit,
        } => to_json_binary(&list_votes(deps, proposal_id, start_after, limit)?),
        QueryMsg::Voter { address } => {
            to_json_binary(&query_voter(deps, address)?)
        }
        QueryMsg::ListVoters { start_after, limit } => {
            to_json_binary(&list_voters(deps, start_after, limit)?)
        }
        QueryMsg::Config {} => to_json_binary(&query_config(deps)?),
    }
}

fn query_threshold(deps: Deps) -> StdResult<ThresholdResponse> {
    let cfg = CONFIG.load(deps.storage)?;
    let total_weight = cfg.group_addr.total_weight(&deps.querier)?;
    Ok(cfg.threshold.to_response(total_weight))
}

fn query_config(deps: Deps) -> StdResult<Config> {
    CONFIG.load(deps.storage)
}

fn query_proposal(deps: Deps, env: Env, id: u64) -> StdResult<ProposalResponse> {
    let prop = PROPOSALS.load(deps.storage, id)?;
    let status = prop.current_status(&env.block);
    let threshold = prop.threshold.to_response(prop.total_weight);
    Ok(ProposalResponse {
        id,
        title: prop.title,
        description: prop.description,
        msgs: prop.msgs,
        status,
        expires: prop.expires,
        proposer: prop.proposer,
        deposit: prop.deposit,
        threshold,
    })
}

// settings for pagination
const MAX_LIMIT: u32 = 30;
const DEFAULT_LIMIT: u32 = 10;

fn list_proposals(
    deps: Deps,
    env: Env,
    start_after: Option<u64>,
    limit: Option<u32>,
) -> StdResult<ProposalListResponse> {
    let limit = limit.unwrap_or(DEFAULT_LIMIT).min(MAX_LIMIT) as usize;
    let start = start_after.map(Bound::exclusive);
    let proposals = PROPOSALS
        .range(deps.storage, start, None, Order::Ascending)
        .take(limit)
        .map(|p| map_proposal(&env.block, p))
        .collect::<StdResult<_>>()?;

    Ok(ProposalListResponse { proposals })
}

fn reverse_proposals(
    deps: Deps,
    env: Env,
    start_before: Option<u64>,
    limit: Option<u32>,
) -> StdResult<ProposalListResponse> {
    let limit = limit.unwrap_or(DEFAULT_LIMIT).min(MAX_LIMIT) as usize;
    let end = start_before.map(Bound::exclusive);
    let props: StdResult<Vec<_>> = PROPOSALS
        .range(deps.storage, None, end, Order::Descending)
        .take(limit)
        .map(|p| map_proposal(&env.block, p))
        .collect();

    Ok(ProposalListResponse { proposals: props? })
}

fn map_proposal(
    block: &BlockInfo,
    item: StdResult<(u64, Proposal)>,
) -> StdResult<ProposalResponse> {
    item.map(|(id, prop)| {
        let status = prop.current_status(block);
        let threshold = prop.threshold.to_response(prop.total_weight);
        ProposalResponse {
            id,
            title: prop.title,
            description: prop.description,
            msgs: prop.msgs,
            status,
            expires: prop.expires,
            deposit: prop.deposit,
            proposer: prop.proposer,
            threshold,
        }
    })
}

fn query_vote(
    deps: Deps,
    proposal_id: u64,
    voter: String,
) -> StdResult<VoteResponse> {
    let voter_addr = deps.api.addr_validate(&voter)?;
    let prop = BALLOTS.may_load(deps.storage, (proposal_id, &voter_addr))?;
    let vote = prop.map(|b| VoteInfo {
        proposal_id,
        voter,
        vote: b.vote,
        weight: b.weight,
    });
    Ok(VoteResponse { vote })
}

fn list_votes(
    deps: Deps,
    proposal_id: u64,
    start_after: Option<String>,
    limit: Option<u32>,
) -> StdResult<VoteListResponse> {
    let limit = limit.unwrap_or(DEFAULT_LIMIT).min(MAX_LIMIT) as usize;
    let addr = maybe_addr(deps.api, start_after)?;
    let start = addr.as_ref().map(Bound::exclusive);

    let votes = BALLOTS
        .prefix(proposal_id)
        .range(deps.storage, start, None, Order::Ascending)
        .take(limit)
        .map(|item| {
            item.map(|(addr, ballot)| VoteInfo {
                proposal_id,
                voter: addr.into(),
                vote: ballot.vote,
                weight: ballot.weight,
            })
        })
        .collect::<StdResult<_>>()?;

    Ok(VoteListResponse { votes })
}

fn query_voter(deps: Deps, voter: String) -> StdResult<VoterResponse> {
    let cfg = CONFIG.load(deps.storage)?;
    let voter_addr = deps.api.addr_validate(&voter)?;
    let weight = cfg.group_addr.is_member(&deps.querier, &voter_addr, None)?;

    Ok(VoterResponse { weight })
}

fn list_voters(
    deps: Deps,
    start_after: Option<String>,
    limit: Option<u32>,
) -> StdResult<VoterListResponse> {
    let cfg = CONFIG.load(deps.storage)?;
    let voters = cfg
        .group_addr
        .list_members(&deps.querier, start_after, limit)?
        .into_iter()
        .map(|member| VoterDetail {
            addr: member.addr,
            weight: member.weight,
        })
        .collect();
    Ok(VoterListResponse { voters })
}

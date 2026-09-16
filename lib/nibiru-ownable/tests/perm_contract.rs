//! Integration tests for a contract that uses the complete perm workflow.
//!
//! The execute entry point calls `assert_msg_auth` once, before
//! dispatch. The ordinary message handlers below do not repeat sender checks.
//! They all increment the same value. A rejected `cw-multi-test` execution
//! therefore proves that the shared generated-policy gate rejected the caller,
//! rather than a check hidden inside the selected handler.

use cosmwasm_schema::{cw_serde, QueryResponses};
use cosmwasm_std::{
    entry_point, to_json_binary, Binary, Deps, DepsMut, Empty, Env, MessageInfo,
    Response, StdError, StdResult,
};
use cw_multi_test::{App, Contract, ContractWrapper, Executor};
use cw_storage_plus::Item;
use nibiru_ownable::{
    assert_msg_auth, get_ownership, initialize_owner, ownable_execute,
    ownable_query, perms_for_members, update_ownership, update_perms, Action,
    MemberPerms, Ownership, OwnershipError, PermError, PermPolicy, PermRule,
    PermUpdate, PermUpdateKind, UserAddr,
};

const VALUE: Item<u64> = Item::new("value");

#[cw_serde]
struct InstantiateMsg {
    owner: String,
}

#[cw_serde]
#[derive(PermPolicy)]
enum NestedMsg {
    #[serde(rename = "record")]
    #[perms(owner_or_any("writer"))]
    Write {},
}

#[ownable_execute(perms)]
#[cw_serde]
#[derive(PermPolicy)]
enum ExecuteMsg {
    #[perms(public)]
    Public {},

    #[perms(owner_or_any())]
    OwnerOnly {},

    #[perms(owner_or_any("writer"))]
    Write {},

    #[perms(owner_or_any("writer", "auditor"))]
    WriteOrAudit {},

    #[perms(nested = "msg")]
    Nested {
        #[serde(rename = "payload")]
        msg: NestedMsg,
    },
}

#[ownable_query(perms)]
#[cw_serde]
#[derive(QueryResponses)]
enum QueryMsg {
    #[returns(u64)]
    Value {},
}

#[derive(thiserror::Error, Debug)]
enum ContractError {
    #[error("{0}")]
    Std(#[from] StdError),
    #[error("{0}")]
    Ownership(#[from] OwnershipError),
    #[error("{0}")]
    Perm(#[from] PermError),
}

#[entry_point]
fn instantiate(
    deps: DepsMut,
    _env: Env,
    _info: MessageInfo,
    msg: InstantiateMsg,
) -> Result<Response, ContractError> {
    initialize_owner(deps.storage, Some(&msg.owner))?;
    VALUE.save(deps.storage, &0)?;
    Ok(Response::new())
}

#[entry_point]
fn execute(
    mut deps: DepsMut,
    env: Env,
    info: MessageInfo,
    msg: ExecuteMsg,
) -> Result<Response, ContractError> {
    // Enforce the generated policy before dispatch. The ordinary branches
    // below deliberately contain no additional sender checks.
    assert_msg_auth(deps.storage, &info.sender, &msg)?;
    match msg {
        ExecuteMsg::UpdateOwnership(action) => {
            update_ownership(
                deps.branch(),
                &env.block,
                info.sender.as_str(),
                action,
            )?;
            Ok(Response::new())
        }
        ExecuteMsg::UpdatePerms(updates) => Ok(Response::new().add_events(
            update_perms::<ExecuteMsg>(deps.storage, &info.sender, updates)?,
        )),
        ExecuteMsg::Public {}
        | ExecuteMsg::OwnerOnly {}
        | ExecuteMsg::Write {}
        | ExecuteMsg::WriteOrAudit {}
        | ExecuteMsg::Nested {
            msg: NestedMsg::Write {},
        } => {
            VALUE.update(deps.storage, |value| -> StdResult<_> {
                Ok(value + 1)
            })?;
            Ok(Response::new())
        }
    }
}

#[entry_point]
fn query(deps: Deps, _env: Env, msg: QueryMsg) -> StdResult<Binary> {
    match msg {
        QueryMsg::Ownership {} => to_json_binary(&get_ownership(deps.storage)?),
        QueryMsg::Perms {} => to_json_binary(&ExecuteMsg::perm_rules()),
        QueryMsg::PermsForMembers { members } => to_json_binary(
            &perms_for_members(deps.storage, members)
                .map_err(|err| StdError::generic_err(err.to_string()))?,
        ),
        QueryMsg::Value {} => to_json_binary(&VALUE.load(deps.storage)?),
    }
}

fn contract() -> Box<dyn Contract<Empty>> {
    Box::new(ContractWrapper::new(execute, instantiate, query))
}

const OWNER: &str = "nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul";
const WRITER: &str = "nibi1k3ddn5w0xm9354w0taup57g5zkzxgewn5ezms0";
const OTHER: &str = "nibi1rgd32wst5dqlmz7q603uzk3c5g9kg584j0d3k2";

fn setup() -> (App, cosmwasm_std::Addr) {
    let mut app = App::default();
    let code = app.store_code(contract());
    let contract = app
        .instantiate_contract(
            code,
            cosmwasm_std::Addr::unchecked(OWNER),
            &InstantiateMsg {
                owner: OWNER.to_string(),
            },
            &[],
            "perm-test",
            None,
        )
        .unwrap();
    (app, contract)
}

fn update(kind: PermUpdateKind, perm: &str, member: &str) -> PermUpdate {
    PermUpdate {
        kind,
        perm: perm.to_string(),
        member: member.parse().unwrap(),
    }
}

/// Confirms that perm-update JSON accepts both address encodings.
///
/// The Bech32 and EVM inputs must resolve to the same `UserAddr`, serialization
/// must use canonical EIP-55 hex, and malformed input must fail to deserialize.
#[test]
fn perm_update_json_accepts_both_user_address_forms() {
    let bech32: ExecuteMsg = serde_json::from_str(
        r#"{"update_perms":[{"kind":"grant","perm":"writer","member":"nibi1gc24lt74ses9swkq6g7cug4e5y72p7e34jqgul"}]}"#,
    )
    .unwrap();
    let evm: ExecuteMsg = serde_json::from_str(
        r#"{"update_perms":[{"kind":"grant","perm":"writer","member":"0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31"}]}"#,
    )
    .unwrap();
    let ExecuteMsg::UpdatePerms(bech32) = bech32 else {
        panic!("expected update_perms")
    };
    let ExecuteMsg::UpdatePerms(evm) = evm else {
        panic!("expected update_perms")
    };

    assert_eq!(bech32[0].member, evm[0].member);
    assert_eq!(
        serde_json::to_value(&bech32[0]).unwrap()["member"],
        "0x46155fAfd58660583ac0d23d8E22B9A13Ca0fb31"
    );
    assert!(serde_json::from_str::<ExecuteMsg>(
        r#"{"update_perms":[{"kind":"grant","perm":"writer","member":"not_an_address"}]}"#,
    )
    .is_err());
}

fn member_perms(
    app: &App,
    contract: &cosmwasm_std::Addr,
    members: Vec<UserAddr>,
) -> Vec<MemberPerms> {
    app.wrap()
        .query_wasm_smart(contract, &QueryMsg::PermsForMembers { members })
        .unwrap()
}

/// Proves that the shared execute gate follows stored membership and ownership.
///
/// The `Write` handler has no sender check of its own. Granting and revoking
/// `writer` therefore controls whether `WRITER` reaches that handler. `OWNER`
/// can always execute it without a stored `writer` membership.
#[test]
fn external_contract_exercises_membership_and_owner_inheritance() {
    let (mut app, contract) = setup();

    // A delegated writer passes the generated `writer` gate. An unrelated
    // account fails before the common value-update handler runs.
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![update(
            PermUpdateKind::Grant,
            "writer",
            WRITER,
        )]),
        &[],
    )
    .unwrap();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(WRITER),
        contract.clone(),
        &ExecuteMsg::Write {},
        &[],
    )
    .unwrap();
    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OTHER),
            contract.clone(),
            &ExecuteMsg::Write {},
            &[],
        )
        .is_err());

    // Revocation removes delegated access, but owner inheritance still passes
    // the same generated gate.
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![update(
            PermUpdateKind::Revoke,
            "writer",
            WRITER,
        )]),
        &[],
    )
    .unwrap();
    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(WRITER),
            contract.clone(),
            &ExecuteMsg::Write {},
            &[],
        )
        .is_err());
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract,
        &ExecuteMsg::Write {},
        &[],
    )
    .unwrap();
}

/// Confirms that discovery queries report the generated policy and memberships.
///
/// The catalog includes owner-only, any-of, nested, and macro-injected routes
/// with the intended JSON field names. The member query returns both address
/// forms and adds `owner` to the owner's stored delegated perms.
#[test]
fn catalog_and_member_queries_match_generated_policy() {
    let (mut app, contract) = setup();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![update(
            PermUpdateKind::Grant,
            "writer",
            OWNER,
        )]),
        &[],
    )
    .unwrap();

    // Query the derive-generated catalog, including the nested route assembled
    // from serialized variant and field names.
    let rules: Vec<PermRule> = app
        .wrap()
        .query_wasm_smart(&contract, &QueryMsg::Perms {})
        .unwrap();
    assert_eq!(
        rules,
        vec![
            PermRule {
                exec_msg: "owner_only".to_string(),
                owner_or_any: vec![],
            },
            PermRule {
                exec_msg: "write".to_string(),
                owner_or_any: vec!["writer".to_string()],
            },
            PermRule {
                exec_msg: "write_or_audit".to_string(),
                owner_or_any: vec!["writer".to_string(), "auditor".to_string(),],
            },
            PermRule {
                exec_msg: "nested.payload.record".to_string(),
                owner_or_any: vec!["writer".to_string()],
            },
            PermRule {
                exec_msg: "update_ownership.transfer_ownership".to_string(),
                owner_or_any: vec![],
            },
            PermRule {
                exec_msg: "update_ownership.renounce_ownership".to_string(),
                owner_or_any: vec![],
            },
            PermRule {
                exec_msg: "update_perms".to_string(),
                owner_or_any: vec![],
            },
        ]
    );
    let catalog_json = serde_json::to_value(&rules).unwrap();
    assert!(catalog_json.as_array().unwrap().iter().all(|rule| {
        let object = rule.as_object().unwrap();
        object.len() == 2
            && object.contains_key("exec_msg")
            && object.contains_key("owner_or_any")
    }));

    // Ownership appears as a virtual perm alongside the stored `writer` perm.
    let owner: UserAddr = OWNER.parse().unwrap();
    let members: Vec<MemberPerms> = app
        .wrap()
        .query_wasm_smart(
            &contract,
            &QueryMsg::PermsForMembers {
                members: vec![owner],
            },
        )
        .unwrap();
    assert_eq!(members[0].member, owner);
    assert_eq!(members[0].member_bech32.as_str(), OWNER);
    assert_eq!(members[0].perms, vec!["owner", "writer"]);
}

/// Proves that a pending owner can reach and accept an ownership transfer.
///
/// `AcceptOwnership` is public only at the shared perm gate. The ownership
/// handler still checks that the sender is the pending owner before changing
/// state.
#[test]
fn pending_owner_can_accept_without_owner_perm_gate() {
    let (mut app, contract) = setup();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdateOwnership(Action::TransferOwnership {
            new_owner: WRITER.to_string(),
            expiry: None,
        }),
        &[],
    )
    .unwrap();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(WRITER),
        contract.clone(),
        &ExecuteMsg::UpdateOwnership(Action::AcceptOwnership),
        &[],
    )
    .unwrap();

    let ownership: Ownership<String> = app
        .wrap()
        .query_wasm_smart(contract, &QueryMsg::Ownership {})
        .unwrap();
    assert_eq!(ownership.owner.as_deref(), Some(WRITER));
}

/// Covers the update batch's owner check, ordering, idempotency, and validation.
///
/// Empty batches are valid. Repeated grants and revokes execute in input order
/// and report whether each item changed storage. An unknown perm invalidates
/// the full batch before its earlier valid update reaches storage.
#[test]
fn batches_are_atomic_ordered_idempotent_and_may_be_empty() {
    let (mut app, contract) = setup();

    // The owner may submit an empty batch. A non-owner may not submit any
    // membership update.
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![]),
        &[],
    )
    .unwrap();
    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OTHER),
            contract.clone(),
            &ExecuteMsg::UpdatePerms(vec![update(
                PermUpdateKind::Grant,
                "writer",
                OTHER,
            )]),
            &[],
        )
        .is_err());

    // Duplicate operations remain valid and expose their no-op status through
    // each event's `changed` attribute.
    let response = app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OWNER),
            contract.clone(),
            &ExecuteMsg::UpdatePerms(vec![
                update(PermUpdateKind::Grant, "writer", WRITER),
                update(PermUpdateKind::Grant, "writer", WRITER),
                update(PermUpdateKind::Revoke, "writer", WRITER),
                update(PermUpdateKind::Revoke, "writer", WRITER),
            ]),
            &[],
        )
        .unwrap();
    let changed: Vec<_> = response
        .events
        .iter()
        .filter(|event| event.ty.ends_with("perm_update"))
        .map(|event| {
            event
                .attributes
                .iter()
                .find(|attribute| attribute.key == "changed")
                .unwrap()
                .value
                .clone()
        })
        .collect();
    assert_eq!(changed, ["true", "false", "true", "false"]);

    // Validation scans the whole batch first. The unknown second perm keeps
    // the valid first grant from being stored.
    let error = app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![
            update(PermUpdateKind::Grant, "writer", WRITER),
            update(PermUpdateKind::Grant, "unknown", OTHER),
        ]),
        &[],
    );
    assert!(error.is_err());
    let members = member_perms(
        &app,
        &contract,
        vec![WRITER.parse().unwrap(), OTHER.parse().unwrap()],
    );
    assert!(members.iter().all(|member| member.perms.is_empty()));
}

/// Exercises public, any-of, query-order, and ownership-transfer semantics.
///
/// An `auditor` may execute an any-of gate but not a writer-only gate. Member
/// queries preserve repeated inputs. After ownership transfer, owner-only
/// access moves to the new owner while delegated memberships remain unchanged.
#[test]
fn public_any_of_queries_and_owner_transfer_keep_their_semantics() {
    let (mut app, contract) = setup();

    // Public execution needs no membership. The same caller needs the matching
    // delegated perm for a gated message.
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OTHER),
        contract.clone(),
        &ExecuteMsg::Public {},
        &[],
    )
    .unwrap();
    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OTHER),
            contract.clone(),
            &ExecuteMsg::Write {},
            &[],
        )
        .is_err());
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdatePerms(vec![update(
            PermUpdateKind::Grant,
            "auditor",
            OTHER,
        )]),
        &[],
    )
    .unwrap();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OTHER),
        contract.clone(),
        &ExecuteMsg::WriteOrAudit {},
        &[],
    )
    .unwrap();
    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OTHER),
            contract.clone(),
            &ExecuteMsg::Write {},
            &[],
        )
        .is_err());

    // Batch queries preserve repeated members and accept an empty input.
    let repeated = member_perms(
        &app,
        &contract,
        vec![OTHER.parse().unwrap(), OTHER.parse().unwrap()],
    );
    assert_eq!(repeated.len(), 2);
    assert_eq!(repeated[0].perms, vec!["auditor"]);
    assert_eq!(repeated[0], repeated[1]);
    assert!(member_perms(&app, &contract, vec![]).is_empty());

    // Ownership inheritance follows the accepted transfer. The unrelated
    // auditor membership remains stored and effective.
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OWNER),
        contract.clone(),
        &ExecuteMsg::UpdateOwnership(Action::TransferOwnership {
            new_owner: WRITER.to_string(),
            expiry: None,
        }),
        &[],
    )
    .unwrap();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(WRITER),
        contract.clone(),
        &ExecuteMsg::UpdateOwnership(Action::AcceptOwnership),
        &[],
    )
    .unwrap();

    assert!(app
        .execute_contract(
            cosmwasm_std::Addr::unchecked(OWNER),
            contract.clone(),
            &ExecuteMsg::OwnerOnly {},
            &[],
        )
        .is_err());
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(WRITER),
        contract.clone(),
        &ExecuteMsg::OwnerOnly {},
        &[],
    )
    .unwrap();
    app.execute_contract(
        cosmwasm_std::Addr::unchecked(OTHER),
        contract.clone(),
        &ExecuteMsg::WriteOrAudit {},
        &[],
    )
    .unwrap();

    let members = member_perms(
        &app,
        &contract,
        vec![
            OWNER.parse().unwrap(),
            WRITER.parse().unwrap(),
            OTHER.parse().unwrap(),
        ],
    );
    assert_eq!(members[0].perms, Vec::<String>::new());
    assert_eq!(members[1].perms, vec!["owner"]);
    assert_eq!(members[2].perms, vec!["auditor"]);
}

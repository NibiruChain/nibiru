use crate::contract::{execute, instantiate, migrate, query, sudo};
use crate::msg::*;
use cosmwasm_std::testing::{
    mock_env, mock_info, MockApi, MockQuerier, MockStorage,
};
use cosmwasm_std::{
    from_json, to_json_binary, ContractResult, CosmosMsg, Empty, Int64,
    OwnedDeps, Querier, QuerierResult, QueryRequest, SystemResult, Uint64,
    WasmMsg,
};
use std::marker::PhantomData;

const ROOT: &str = easy_addr::addr!("root");
const GOV: &str = easy_addr::addr!("governance");
const OTHER: &str = easy_addr::addr!("other");
/// Answers the native sudo query with a mutable root and extra role fields,
/// exercising live authority lookup and tolerant decoding of module responses.
struct RootQuerier {
    root: String,
    base: MockQuerier,
}
impl Querier for RootQuerier {
    fn raw_query(&self, bytes: &[u8]) -> QuerierResult {
        let request: QueryRequest<Empty> = from_json(bytes).unwrap();
        if let QueryRequest::Stargate { path, .. } = &request {
            if path == nibiru_std::sudo::QUERY_SUDOERS_PATH {
                return SystemResult::Ok(ContractResult::Ok(to_json_binary(&serde_json::json!({"sudoers":{"root":self.root,"roles":[]}})).unwrap()));
            }
        }
        self.base.raw_query(bytes)
    }
}
type Deps = OwnedDeps<MockStorage, MockApi, RootQuerier>;
/// Creates an absent legacy state so tests distinguish missing values from zero.
fn empty_snapshot() -> Snapshot {
    Snapshot {
        plan: None,
        completed: vec![],
        module_versions: vec![],
        protocol_version: None,
    }
}
/// Instantiates with a fixed governance authority and a separately mutable root.
fn setup(state: Snapshot) -> Deps {
    let mut deps = OwnedDeps {
        storage: MockStorage::default(),
        api: MockApi::default(),
        querier: RootQuerier {
            root: ROOT.into(),
            base: MockQuerier::default(),
        },
        custom_query_type: PhantomData,
    };
    instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info(OTHER, &[]),
        InstantiateMsg {
            gov_authority: GOV.into(),
            state,
        },
    )
    .unwrap();
    deps
}
/// Builds one scheduling request with explicit block or Unix-nanosecond inputs.
fn req(height: i64, target_time_unix_nanos: Option<u64>) -> ScheduleRequest {
    ScheduleRequest {
        name: "vNext".into(),
        block_height: Int64::new(height),
        target_time_unix_nanos: target_time_unix_nanos.map(Uint64::new),
        info: "release".into(),
    }
}
/// Decodes the real query entry point instead of reading internal storage directly.
fn read<T: serde::de::DeserializeOwned>(deps: &Deps, msg: QueryMsg) -> T {
    from_json(query(deps.as_ref(), mock_env(), msg).unwrap()).unwrap()
}
/// Preserves all four imported state families and the absent-versus-zero
/// protocol distinction needed for native migration and genesis compatibility.
#[test]
fn import_preserves_all_families_and_absence() {
    let snapshot = Snapshot {
        plan: Some(Plan {
            name: "executing".into(),
            block_height: Int64::new(1),
            info: "old".into(),
        }),
        completed: vec![CompletedUpgrade {
            name: "old".into(),
            block_height: Int64::new(1),
        }],
        module_versions: vec![ModuleVersion {
            name: "wasm".into(),
            version: Uint64::new(4),
        }],
        protocol_version: None,
    };
    let deps = setup(snapshot.clone());
    assert_eq!(read::<Snapshot>(&deps, QueryMsg::Snapshot {}), snapshot);
    let mut deps = setup(empty_snapshot());
    sudo(
        deps.as_mut(),
        mock_env(),
        SudoMsg::SetProtocolVersion {
            version: Uint64::zero(),
        },
    )
    .unwrap();
    assert_eq!(
        read::<Option<Uint64>>(&deps, QueryMsg::ProtocolVersion {}),
        Some(Uint64::zero())
    );
}
/// Scheduling and cancellation require the native governance account, even
/// when the caller is the sudo root that controls contract configuration.
#[test]
fn governance_gate_does_not_accept_root_or_claimed_owner() {
    let mut deps = setup(empty_snapshot());
    let env = mock_env();
    for sender in [ROOT, OTHER] {
        assert!(execute(
            deps.as_mut(),
            env.clone(),
            mock_info(sender, &[]),
            ExecuteMsg::Schedule {
                plan: req(env.block.height as i64 + 100, None)
            }
        )
        .is_err());
    }
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::Schedule {
            plan: req(env.block.height as i64 + 100, None),
        },
    )
    .unwrap();
    assert!(execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::Cancel {}
    )
    .is_err());
    execute(
        deps.as_mut(),
        env,
        mock_info(GOV, &[]),
        ExecuteMsg::Cancel {},
    )
    .unwrap();
    assert_eq!(read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}), None);
}
/// Waits for the exact nanosecond target, samples policy when used, and keeps
/// the selected execution block fixed after later configuration changes.
#[test]
fn target_selection_samples_delay_once_and_preserves_full_integers() {
    let mut deps = setup(empty_snapshot());
    let mut env = mock_env();
    let target = env.block.time.nanos() + 1;
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::Schedule {
            plan: req(0, Some(target)),
        },
    )
    .unwrap();
    assert_eq!(read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}), None);
    let pending: TimeTarget =
        read::<Option<TimeTarget>>(&deps, QueryMsg::TimeTarget {}).unwrap();
    assert_eq!(pending.target_time_unix_nanos.u64(), target);
    sudo(
        deps.as_mut(),
        env.clone(),
        SudoMsg::SelectExecutionHeight {},
    )
    .unwrap();
    assert_eq!(read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}), None);
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::UpdateConfig {
            activation_delay_blocks: Uint64::new(2),
        },
    )
    .unwrap();
    env.block.time = env.block.time.plus_nanos(1);
    sudo(
        deps.as_mut(),
        env.clone(),
        SudoMsg::SelectExecutionHeight {},
    )
    .unwrap();
    let selected =
        read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}).unwrap();
    assert_eq!(selected.block_height.i64(), env.block.height as i64 + 2);
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::UpdateConfig {
            activation_delay_blocks: Uint64::new(2000),
        },
    )
    .unwrap();
    env.block.height += 1;
    sudo(deps.as_mut(), env, SudoMsg::SelectExecutionHeight {}).unwrap();
    assert_eq!(
        read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}).unwrap(),
        selected
    );
}
/// Rejects malformed, elapsed, or already-completed scheduling requests
/// without replacing an existing authorized pending request.
#[test]
fn rejects_ambiguous_overdue_and_completed_requests_without_replacement() {
    let mut deps = setup(empty_snapshot());
    let env = mock_env();
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::Schedule {
            plan: req(env.block.height as i64 + 100, None),
        },
    )
    .unwrap();
    let old = read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {});
    for plan in [
        req(0, None),
        req(-1, None),
        req(0, Some(env.block.time.nanos())),
        req(100, Some(env.block.time.nanos() + 1)),
    ] {
        assert!(execute(
            deps.as_mut(),
            env.clone(),
            mock_info(GOV, &[]),
            ExecuteMsg::Schedule { plan }
        )
        .is_err());
        assert_eq!(read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}), old);
    }
    sudo(
        deps.as_mut(),
        env.clone(),
        SudoMsg::RecordCompleted {
            name: "vNext".into(),
        },
    )
    .unwrap();
    assert!(execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::Schedule {
            plan: req(env.block.height as i64 + 100, None)
        }
    )
    .is_err());
}
/// Preserves legacy duplicate names, indexes name lookups to the earliest
/// completion, and pages tied block heights without omitting or repeating records.
#[test]
fn history_preserves_duplicate_names_height_ties_and_pagination() {
    let records = vec![
        CompletedUpgrade {
            name: "same".into(),
            block_height: Int64::new(10),
        },
        CompletedUpgrade {
            name: "same".into(),
            block_height: Int64::new(20),
        },
        CompletedUpgrade {
            name: "zzz".into(),
            block_height: Int64::new(20),
        },
    ];
    let mut snapshot = empty_snapshot();
    snapshot.completed = records.clone();
    let deps = setup(snapshot);
    assert_eq!(
        read::<Int64>(
            &deps,
            QueryMsg::Applied {
                name: "same".into()
            }
        )
        .i64(),
        10
    );
    assert_eq!(
        read::<Option<CompletedUpgrade>>(&deps, QueryMsg::LastCompleted {}),
        Some(records[2].clone())
    );
    let page: Vec<CompletedUpgrade> = read(
        &deps,
        QueryMsg::History {
            start_after: Some(records[0].clone()),
            limit: Some(1),
            descending: Some(false),
        },
    );
    assert_eq!(page, vec![records[1].clone()]);
    let reverse: Vec<CompletedUpgrade> = read(
        &deps,
        QueryMsg::History {
            start_after: Some(records[2].clone()),
            limit: Some(50),
            descending: Some(true),
        },
    );
    assert_eq!(reverse, vec![records[1].clone(), records[0].clone()]);
}
/// Native migrations update selected module versions; omitted modules must
/// survive, and query results remain ordered deterministically by module name.
#[test]
fn version_updates_merge_and_are_ordered() {
    let mut deps = setup(empty_snapshot());
    for versions in [
        vec![
            ModuleVersion {
                name: "z".into(),
                version: Uint64::new(9),
            },
            ModuleVersion {
                name: "a".into(),
                version: Uint64::new(1),
            },
        ],
        vec![ModuleVersion {
            name: "a".into(),
            version: Uint64::new(2),
        }],
    ] {
        sudo(
            deps.as_mut(),
            mock_env(),
            SudoMsg::SetModuleVersions { versions },
        )
        .unwrap();
    }
    let records: Vec<ModuleVersion> =
        read(&deps, QueryMsg::ModuleVersions { name: None });
    assert_eq!(
        records
            .iter()
            .map(|v| (v.name.as_str(), v.version.u64()))
            .collect::<Vec<_>>(),
        vec![("a", 2), ("z", 9)]
    );
}
/// Runtime policy and self-migration follow the live root after rotation;
/// governance scheduling authority does not grant those administrative rights.
#[test]
fn root_rotation_controls_config_and_self_migration() {
    let mut deps = setup(empty_snapshot());
    let env = mock_env();
    assert!(execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::UpdateConfig {
            activation_delay_blocks: Uint64::new(2)
        }
    )
    .is_err());
    assert!(execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::UpdateConfig {
            activation_delay_blocks: Uint64::new(1)
        }
    )
    .is_err());
    deps.querier.root = OTHER.into();
    assert!(execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::Migrate {
            code_id: 42,
            msg: to_json_binary(&MigrateMsg {}).unwrap()
        }
    )
    .is_err());
    let response = execute(
        deps.as_mut(),
        env.clone(),
        mock_info(OTHER, &[]),
        ExecuteMsg::Migrate {
            code_id: 42,
            msg: to_json_binary(&MigrateMsg {}).unwrap(),
        },
    )
    .unwrap();
    assert!(
        matches!(&response.messages[0].msg,CosmosMsg::Wasm(WasmMsg::Migrate{contract_addr,new_code_id:42,..}) if contract_addr==env.contract.address.as_str())
    );
    let before = read::<Snapshot>(&deps, QueryMsg::Snapshot {});
    migrate(deps.as_mut(), env, MigrateMsg {}).unwrap();
    assert_eq!(read::<Snapshot>(&deps, QueryMsg::Snapshot {}), before);
}
/// Replacement gets a distinct identity for native failure suppression. A
/// height overflow must not publish a partially selected execution block.
#[test]
fn replacement_assigns_a_new_id_and_overflow_is_atomic() {
    let mut deps = setup(empty_snapshot());
    let mut env = mock_env();
    let target = env.block.time.nanos() + 1;
    for _ in 0..2 {
        execute(
            deps.as_mut(),
            env.clone(),
            mock_info(GOV, &[]),
            ExecuteMsg::Schedule {
                plan: req(0, Some(target)),
            },
        )
        .unwrap();
    }
    let before =
        read::<Option<PendingSchedule>>(&deps, QueryMsg::Status {}).unwrap();
    assert_eq!(before.id.u64(), 3);
    env.block.height = i64::MAX as u64;
    env.block.time = env.block.time.plus_nanos(1);
    assert!(sudo(deps.as_mut(), env, SudoMsg::SelectExecutionHeight {}).is_err());
    assert_eq!(
        read::<Option<PendingSchedule>>(&deps, QueryMsg::Status {}).unwrap(),
        before
    );
}

/// Corrupted or improperly migrated policy cannot bypass the minimum warning
/// interval and select an immediate upgrade, even if execute validation was bypassed.
#[test]
fn invalid_imported_delay_cannot_select_a_same_block_height() {
    let mut deps = setup(empty_snapshot());
    let mut env = mock_env();
    let target = env.block.time.nanos() + 1;
    execute(
        deps.as_mut(),
        env.clone(),
        mock_info(GOV, &[]),
        ExecuteMsg::Schedule {
            plan: req(0, Some(target)),
        },
    )
    .unwrap();
    crate::state::CONFIG
        .update(
            &mut deps.storage,
            |mut config| -> cosmwasm_std::StdResult<_> {
                config.activation_delay_blocks = Uint64::new(1);
                Ok(config)
            },
        )
        .unwrap();
    env.block.time = env.block.time.plus_nanos(1);
    assert!(sudo(deps.as_mut(), env, SudoMsg::SelectExecutionHeight {}).is_err());
    assert_eq!(read::<Option<Plan>>(&deps, QueryMsg::CurrentPlan {}), None);
}

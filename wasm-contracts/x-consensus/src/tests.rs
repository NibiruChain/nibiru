//! tests.rs: Unit tests for x-consensus entry points and validation.

use std::marker::PhantomData;

use cosmwasm_std::testing::{
    mock_env, mock_info, MockApi, MockQuerier, MockStorage,
};
use cosmwasm_std::{
    from_json, to_json_binary, Binary, ContractResult, CosmosMsg, Empty, Int64,
    OwnedDeps, Querier, QuerierResult, QueryRequest, SystemError, SystemResult,
    WasmMsg,
};

use crate::contract::{execute, instantiate, migrate, query};
use crate::error::ContractError;
use crate::msg::{
    BlockParams, ConsensusParams, EvidenceParams, ExecuteMsg, InstantiateMsg,
    MigrateMsg, QueryMsg, ValidatorParams, MAX_MIN_BLOCK_COMMIT_DURATION_MS,
};
use crate::state::PARAMS;
use crate::sudo::QUERY_SUDOERS_PATH;
use crate::validate::MAX_BLOCK_SIZE_BYTES;

/// x/sudo root address made by the default mock API.
const ROOT: &str = easy_addr::addr!("root");
/// A different valid address with no rights.
const NOT_ROOT: &str = easy_addr::addr!("not-root");

/// Fixture shared with the Go package `x/xconsensus`.
const INSTANTIATE_FIXTURE: &str =
    include_str!("../testdata/instantiate_msg.json");
/// Fixture shared with the Go package `x/xconsensus`.
const UPDATE_PARAMS_FIXTURE: &str =
    include_str!("../testdata/update_params_msg.json");

/// Querier that answers the x/sudo `QuerySudoers` Stargate query like the
/// chain does, and delegates every other request to [`MockQuerier`].
struct SudoQuerier {
    /// Root returned by `QuerySudoers`. Empty means no root is set.
    root: String,
    /// When true, `QuerySudoers` fails as if the path were not accepted.
    fail: bool,
    /// Handles all other requests.
    base: MockQuerier,
}

impl Querier for SudoQuerier {
    fn raw_query(&self, bin_request: &[u8]) -> QuerierResult {
        let request: QueryRequest<Empty> = match from_json(bin_request) {
            Ok(r) => r,
            Err(e) => {
                return SystemResult::Err(SystemError::InvalidRequest {
                    error: e.to_string(),
                    request: bin_request.into(),
                })
            }
        };
        match request {
            QueryRequest::Stargate { path, .. }
                if path == QUERY_SUDOERS_PATH =>
            {
                if self.fail {
                    return SystemResult::Err(SystemError::UnsupportedRequest {
                        kind: path,
                    });
                }
                // Same shape as the Wasm keeper's proto JSON, including the
                // roles field the contract ignores.
                let json = serde_json::json!({
                    "sudoers": {
                        "root": self.root,
                        "roles": [{"role": "wasm_deployer", "members": []}],
                    }
                });
                SystemResult::Ok(ContractResult::Ok(
                    to_json_binary(&json).unwrap(),
                ))
            }
            _ => self.base.raw_query(bin_request),
        }
    }
}

type Deps = OwnedDeps<MockStorage, MockApi, SudoQuerier>;

/// Go `DefaultConsensusParams` (`bft/types/params.go`) without the version,
/// and with the commit duration unset.
fn default_params() -> ConsensusParams {
    ConsensusParams {
        block: BlockParams {
            max_bytes: Int64::new(22_020_096),
            max_gas: Int64::new(-1),
        },
        evidence: EvidenceParams {
            max_age_num_blocks: Int64::new(100_000),
            max_age_duration_ns: Int64::new(48 * 3600 * 1_000_000_000),
            max_bytes: Int64::new(1_048_576),
        },
        validator: ValidatorParams {
            pub_key_types: vec!["ed25519".to_string()],
        },
        min_block_commit_duration_ms: None,
    }
}

/// Fresh dependencies whose x/sudo root is `root`.
fn deps_with_root(root: &str) -> Deps {
    OwnedDeps {
        storage: MockStorage::default(),
        api: MockApi::default(),
        querier: SudoQuerier {
            root: root.to_string(),
            fail: false,
            base: MockQuerier::default(),
        },
        custom_query_type: PhantomData,
    }
}

/// Instantiates the contract with `params` and x/sudo root `ROOT`.
fn setup(params: ConsensusParams) -> Deps {
    let mut deps = deps_with_root(ROOT);
    instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info(NOT_ROOT, &[]),
        InstantiateMsg { params },
    )
    .expect("instantiate");
    deps
}

/// Queries the stored params.
fn query_params(deps: &Deps) -> ConsensusParams {
    from_json(query(deps.as_ref(), mock_env(), QueryMsg::Params {}).unwrap())
        .unwrap()
}

/// Sends `UpdateParams` from `sender`.
fn update(
    deps: &mut Deps,
    sender: &str,
    params: ConsensusParams,
) -> Result<cosmwasm_std::Response, ContractError> {
    execute(
        deps.as_mut(),
        mock_env(),
        mock_info(sender, &[]),
        ExecuteMsg::UpdateParams { params },
    )
}

/// Instantiate stores the params exactly as given. The Go module seeds the
/// contract from x/consensus this way, so any drift would change the chain.
#[test]
fn instantiate_stores_params() {
    let deps = setup(default_params());
    assert_eq!(query_params(&deps), default_params());
}

/// Instantiate works with no x/sudo root and a failing root query, so a new
/// chain can create x-consensus at genesis before any root exists.
#[test]
fn instantiate_needs_no_sudo_root() {
    let mut deps = deps_with_root("");
    deps.querier.fail = true;
    instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info(NOT_ROOT, &[]),
        InstantiateMsg {
            params: default_params(),
        },
    )
    .unwrap();
    assert_eq!(query_params(&deps), default_params());
}

/// Instantiate runs the same validation as an update and stores nothing on
/// failure, so the contract never starts with params Nibiru BFT rejects.
#[test]
fn instantiate_rejects_invalid_params() {
    let mut deps = deps_with_root(ROOT);
    let mut params = default_params();
    params.block.max_bytes = Int64::new(0);
    let err = instantiate(
        deps.as_mut(),
        mock_env(),
        mock_info(ROOT, &[]),
        InstantiateMsg { params },
    )
    .unwrap_err();
    assert!(matches!(err, ContractError::InvalidParams(_)), "{err}");
    assert!(PARAMS.may_load(deps.as_ref().storage).unwrap().is_none());
}

/// The root can change every group at once, and setting the commit wait back
/// to `None` unsets it, which hands pacing back to each node's config.
#[test]
fn update_params_round_trip() {
    let mut deps = setup(default_params());
    let mut params = default_params();
    params.block.max_gas = Int64::new(150_000_000);
    params.validator.pub_key_types =
        vec!["ed25519".to_string(), "secp256k1".to_string()];
    params.min_block_commit_duration_ms = Some(200);
    update(&mut deps, ROOT, params.clone()).unwrap();
    assert_eq!(query_params(&deps), params);

    // Setting the value back to None unsets it.
    params.min_block_commit_duration_ms = None;
    update(&mut deps, ROOT, params.clone()).unwrap();
    assert_eq!(query_params(&deps).min_block_commit_duration_ms, None);
}

/// A rejected update leaves the old params in place, so a bad proposal can
/// never leave the contract half updated.
#[test]
fn update_params_rejects_invalid_and_keeps_old_value() {
    let mut deps = setup(default_params());
    let mut params = default_params();
    params.min_block_commit_duration_ms =
        Some(MAX_MIN_BLOCK_COMMIT_DURATION_MS + 1);
    let err = update(&mut deps, ROOT, params).unwrap_err();
    assert!(matches!(err, ContractError::InvalidParams(_)), "{err}");
    assert_eq!(query_params(&deps), default_params());
}

/// The live x/sudo root is the only writer: non-root senders, an empty root
/// and a failed root query are all rejected, and a root rotation moves the
/// right at once. The root check also runs before validation.
#[test]
fn sudo_root_checks() {
    let mut deps = setup(default_params());
    let mut changed = default_params();
    changed.min_block_commit_duration_ms = Some(1_000);

    // A non-root sender is rejected.
    let err = update(&mut deps, NOT_ROOT, changed.clone()).unwrap_err();
    assert_eq!(
        err,
        ContractError::Unauthorized {
            sender: NOT_ROOT.to_string(),
            root: ROOT.to_string(),
        }
    );

    // The root check runs before validation.
    let mut bad = default_params();
    bad.validator.pub_key_types = vec![];
    let err = update(&mut deps, NOT_ROOT, bad).unwrap_err();
    assert!(matches!(err, ContractError::Unauthorized { .. }), "{err}");

    // An empty root rejects everyone, including a sender equal to "".
    deps.querier.root = String::new();
    let err = update(&mut deps, ROOT, changed.clone()).unwrap_err();
    assert_eq!(err, ContractError::NoSudoRoot);

    // A failed root query rejects the update.
    deps.querier.root = ROOT.to_string();
    deps.querier.fail = true;
    let err = update(&mut deps, ROOT, changed.clone()).unwrap_err();
    assert!(matches!(err, ContractError::Std(_)), "{err}");
    assert_eq!(query_params(&deps), default_params());

    // Root rotation in x/sudo takes effect immediately: the old root loses
    // rights and the new root gains them.
    deps.querier.fail = false;
    deps.querier.root = NOT_ROOT.to_string();
    let err = update(&mut deps, ROOT, changed.clone()).unwrap_err();
    assert!(matches!(err, ContractError::Unauthorized { .. }), "{err}");
    update(&mut deps, NOT_ROOT, changed.clone()).unwrap();
    assert_eq!(query_params(&deps), changed);
}

/// Migrate sends exactly one Wasm migrate for the contract's own address,
/// with the caller's code ID and message, and leaves the params alone.
#[test]
fn migrate_dispatches_self_migration() {
    let mut deps = setup(default_params());
    let env = mock_env();
    let msg = Binary::from(br#"{"note":"v2"}"#);
    let res = execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::Migrate {
            code_id: 7,
            msg: msg.clone(),
        },
    )
    .unwrap();
    assert_eq!(res.messages.len(), 1);
    assert_eq!(
        res.messages[0].msg,
        CosmosMsg::Wasm(WasmMsg::Migrate {
            contract_addr: env.contract.address.to_string(),
            new_code_id: 7,
            msg,
        })
    );
    assert_eq!(query_params(&deps), default_params());
}

/// UpdateAdmin sends exactly one Wasm admin change for the contract's own
/// address. This is how the root moves migration rights elsewhere.
#[test]
fn update_admin_dispatches_admin_change() {
    let mut deps = setup(default_params());
    let env = mock_env();
    let res = execute(
        deps.as_mut(),
        env.clone(),
        mock_info(ROOT, &[]),
        ExecuteMsg::UpdateAdmin {
            admin: NOT_ROOT.to_string(),
        },
    )
    .unwrap();
    assert_eq!(res.messages.len(), 1);
    assert_eq!(
        res.messages[0].msg,
        CosmosMsg::Wasm(WasmMsg::UpdateAdmin {
            contract_addr: env.contract.address.to_string(),
            admin: NOT_ROOT.to_string(),
        })
    );
}

/// UpdateAdmin rejects an address that does not validate (here, not
/// normalized), so a malformed value fails before any message is sent.
#[test]
fn update_admin_rejects_invalid_address() {
    let mut deps = setup(default_params());
    let err = execute(
        deps.as_mut(),
        mock_env(),
        mock_info(ROOT, &[]),
        ExecuteMsg::UpdateAdmin {
            admin: NOT_ROOT.to_uppercase(),
        },
    )
    .unwrap_err();
    assert!(matches!(err, ContractError::Std(_)), "{err}");
}

/// Migrate and UpdateAdmin share the root check with UpdateParams. Without
/// it, anyone could migrate the contract, since the contract is its own admin.
#[test]
fn admin_messages_are_root_only() {
    let mut deps = setup(default_params());
    let msgs = [
        ExecuteMsg::Migrate {
            code_id: 7,
            msg: Binary::from(b"{}"),
        },
        ExecuteMsg::UpdateAdmin {
            admin: NOT_ROOT.to_string(),
        },
        // The root check runs before the admin address is validated.
        ExecuteMsg::UpdateAdmin {
            admin: NOT_ROOT.to_uppercase(),
        },
    ];
    for msg in msgs {
        let err = execute(
            deps.as_mut(),
            mock_env(),
            mock_info(NOT_ROOT, &[]),
            msg.clone(),
        )
        .unwrap_err();
        assert!(
            matches!(err, ContractError::Unauthorized { .. }),
            "{msg:?}: {err}"
        );
    }
    deps.querier.root = String::new();
    let err = execute(
        deps.as_mut(),
        mock_env(),
        mock_info(ROOT, &[]),
        ExecuteMsg::Migrate {
            code_id: 7,
            msg: Binary::from(b"{}"),
        },
    )
    .unwrap_err();
    assert_eq!(err, ContractError::NoSudoRoot);
}

/// The migrate entry point keeps the params, so migrating into this code
/// never resets consensus params.
#[test]
fn migrate_keeps_state() {
    let mut deps = setup(default_params());
    migrate(deps.as_mut(), mock_env(), MigrateMsg {}).unwrap();
    assert_eq!(query_params(&deps), default_params());
}

/// One validation case: a mutation of the default params and the expected
/// result (`None` for valid, or a substring of the error).
struct Case {
    name: &'static str,
    mutate: fn(&mut ConsensusParams),
    err: Option<&'static str>,
}

/// Each rule from `bft/types/params.go` plus the commit wait cap, at and just
/// past its boundary, both as a pure check and through the contract. Keeps
/// the contract from accepting a value Nibiru BFT would reject, or the reverse.
#[test]
fn validation_boundaries() {
    let cases = [
        // block.max_bytes in (0, MAX_BLOCK_SIZE_BYTES]
        Case {
            name: "block.max_bytes 0",
            mutate: |p| p.block.max_bytes = Int64::new(0),
            err: Some("block.max_bytes must be greater than 0"),
        },
        Case {
            name: "block.max_bytes -1",
            mutate: |p| p.block.max_bytes = Int64::new(-1),
            err: Some("block.max_bytes must be greater than 0"),
        },
        Case {
            name: "block.max_bytes 1 (evidence.max_bytes 0)",
            mutate: |p| {
                p.block.max_bytes = Int64::new(1);
                p.evidence.max_bytes = Int64::new(0);
            },
            err: None,
        },
        Case {
            name: "block.max_bytes at max",
            mutate: |p| p.block.max_bytes = Int64::new(MAX_BLOCK_SIZE_BYTES),
            err: None,
        },
        Case {
            name: "block.max_bytes above max",
            mutate: |p| p.block.max_bytes = Int64::new(MAX_BLOCK_SIZE_BYTES + 1),
            err: Some("block.max_bytes is too big"),
        },
        // block.max_gas >= -1
        Case {
            name: "block.max_gas -1",
            mutate: |p| p.block.max_gas = Int64::new(-1),
            err: None,
        },
        Case {
            name: "block.max_gas 0",
            mutate: |p| p.block.max_gas = Int64::new(0),
            err: None,
        },
        Case {
            name: "block.max_gas i64 max",
            mutate: |p| p.block.max_gas = Int64::new(i64::MAX),
            err: None,
        },
        Case {
            name: "block.max_gas -2",
            mutate: |p| p.block.max_gas = Int64::new(-2),
            err: Some("block.max_gas must be greater or equal to -1"),
        },
        // evidence.max_age_num_blocks > 0
        Case {
            name: "evidence.max_age_num_blocks 0",
            mutate: |p| p.evidence.max_age_num_blocks = Int64::new(0),
            err: Some("evidence.max_age_num_blocks must be greater than 0"),
        },
        Case {
            name: "evidence.max_age_num_blocks 1",
            mutate: |p| p.evidence.max_age_num_blocks = Int64::new(1),
            err: None,
        },
        // evidence.max_age_duration_ns > 0
        Case {
            name: "evidence.max_age_duration_ns 0",
            mutate: |p| p.evidence.max_age_duration_ns = Int64::new(0),
            err: Some("evidence.max_age_duration_ns must be greater than 0"),
        },
        Case {
            name: "evidence.max_age_duration_ns -1",
            mutate: |p| p.evidence.max_age_duration_ns = Int64::new(-1),
            err: Some("evidence.max_age_duration_ns must be greater than 0"),
        },
        Case {
            name: "evidence.max_age_duration_ns 1",
            mutate: |p| p.evidence.max_age_duration_ns = Int64::new(1),
            err: None,
        },
        Case {
            name: "evidence.max_age_duration_ns i64 max",
            mutate: |p| p.evidence.max_age_duration_ns = Int64::new(i64::MAX),
            err: None,
        },
        // evidence.max_bytes in [0, block.max_bytes]
        Case {
            name: "evidence.max_bytes equals block.max_bytes",
            mutate: |p| p.evidence.max_bytes = p.block.max_bytes,
            err: None,
        },
        Case {
            name: "evidence.max_bytes above block.max_bytes",
            mutate: |p| {
                p.evidence.max_bytes = Int64::new(p.block.max_bytes.i64() + 1)
            },
            err: Some("evidence.max_bytes is greater than upper bound"),
        },
        Case {
            name: "evidence.max_bytes 0",
            mutate: |p| p.evidence.max_bytes = Int64::new(0),
            err: None,
        },
        Case {
            name: "evidence.max_bytes -1",
            mutate: |p| p.evidence.max_bytes = Int64::new(-1),
            err: Some("evidence.max_bytes must be non negative"),
        },
        // validator.pub_key_types non-empty and known
        Case {
            name: "pub_key_types empty",
            mutate: |p| p.validator.pub_key_types = vec![],
            err: Some("len(validator.pub_key_types) must be greater than 0"),
        },
        Case {
            name: "pub_key_types secp256k1 only",
            mutate: |p| {
                p.validator.pub_key_types = vec!["secp256k1".to_string()]
            },
            err: None,
        },
        Case {
            name: "pub_key_types duplicates allowed as in Go",
            mutate: |p| {
                p.validator.pub_key_types =
                    vec!["ed25519".to_string(), "ed25519".to_string()]
            },
            err: None,
        },
        Case {
            name: "pub_key_types unknown",
            mutate: |p| {
                p.validator.pub_key_types =
                    vec!["ed25519".to_string(), "sr25519".to_string()]
            },
            err: Some("validator.pub_key_types[1], sr25519, is an unknown"),
        },
        Case {
            name: "pub_key_types Amino name is not an ABCI name",
            mutate: |p| {
                p.validator.pub_key_types =
                    vec!["tendermint/PubKeyEd25519".to_string()]
            },
            err: Some("is an unknown pubkey type"),
        },
        // min_block_commit_duration_ms in [0, 30000] or unset
        Case {
            name: "min_block_commit_duration_ms unset",
            mutate: |p| p.min_block_commit_duration_ms = None,
            err: None,
        },
        Case {
            name: "min_block_commit_duration_ms 0",
            mutate: |p| p.min_block_commit_duration_ms = Some(0),
            err: None,
        },
        Case {
            name: "min_block_commit_duration_ms 30000",
            mutate: |p| p.min_block_commit_duration_ms = Some(30_000),
            err: None,
        },
        Case {
            name: "min_block_commit_duration_ms 30001",
            mutate: |p| p.min_block_commit_duration_ms = Some(30_001),
            err: Some("min_block_commit_duration_ms must be at most 30000"),
        },
        Case {
            name: "min_block_commit_duration_ms u64 max",
            mutate: |p| p.min_block_commit_duration_ms = Some(u64::MAX),
            err: Some("min_block_commit_duration_ms must be at most 30000"),
        },
    ];

    for case in cases {
        let mut params = default_params();
        (case.mutate)(&mut params);

        // Pure validation.
        let got = params.validate();
        // The same rule through the contract.
        let mut deps = setup(default_params());
        let via_exec = update(&mut deps, ROOT, params.clone());

        match case.err {
            None => {
                assert!(got.is_ok(), "{}: {:?}", case.name, got);
                assert!(via_exec.is_ok(), "{}: {:?}", case.name, via_exec);
                assert_eq!(query_params(&deps), params, "{}", case.name);
            }
            Some(want) => {
                let err = got.expect_err(case.name).to_string();
                assert!(err.contains(want), "{}: got {err}", case.name);
                let err = via_exec.expect_err(case.name).to_string();
                assert!(err.contains(want), "{}: got {err}", case.name);
                assert_eq!(query_params(&deps), default_params());
            }
        }
    }
}

/// The fixtures are the JSON contract with the Go package, which reads the
/// same files. They must decode to the expected values and re-encode to the
/// same JSON, and the JSON conventions (strings for int64, null or absent for
/// unset, no unknown fields) must hold.
#[test]
fn json_fixtures_match_types() {
    let msg: InstantiateMsg = serde_json::from_str(INSTANTIATE_FIXTURE).unwrap();
    assert_eq!(msg.params, default_params());
    let want: serde_json::Value =
        serde_json::from_str(INSTANTIATE_FIXTURE).unwrap();
    assert_eq!(serde_json::to_value(&msg).unwrap(), want);

    let msg: ExecuteMsg = serde_json::from_str(UPDATE_PARAMS_FIXTURE).unwrap();
    let mut expected = default_params();
    expected.validator.pub_key_types =
        vec!["ed25519".to_string(), "secp256k1".to_string()];
    expected.min_block_commit_duration_ms = Some(200);
    assert_eq!(
        msg,
        ExecuteMsg::UpdateParams {
            params: expected.clone(),
        }
    );
    let want: serde_json::Value =
        serde_json::from_str(UPDATE_PARAMS_FIXTURE).unwrap();
    assert_eq!(serde_json::to_value(&msg).unwrap(), want);

    // A missing field decodes as unset, same as null.
    let mut v: serde_json::Value =
        serde_json::from_str(INSTANTIATE_FIXTURE).unwrap();
    v["params"]
        .as_object_mut()
        .unwrap()
        .remove("min_block_commit_duration_ms");
    let msg: InstantiateMsg = serde_json::from_value(v).unwrap();
    assert_eq!(msg.params.min_block_commit_duration_ms, None);

    // A JSON number where a string is expected is rejected, so a decoder
    // cannot silently round a large int64.
    let bad = INSTANTIATE_FIXTURE.replace("\"22020096\"", "22020096");
    assert!(serde_json::from_str::<InstantiateMsg>(&bad).is_err());

    // The contract has no power over the app version: a params object that
    // still carries one is rejected instead of silently dropped.
    let mut v: serde_json::Value =
        serde_json::from_str(INSTANTIATE_FIXTURE).unwrap();
    v["params"]["version"] = serde_json::json!({"app": "1"});
    assert!(serde_json::from_value::<InstantiateMsg>(v).is_err());
}

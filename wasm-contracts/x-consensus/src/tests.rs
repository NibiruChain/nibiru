//! tests.rs: Unit tests for x-consensus entry points and validation.

use cosmwasm_std::testing::{
    mock_dependencies, mock_env, mock_info, MockApi, MockQuerier, MockStorage,
};
use cosmwasm_std::{from_json, Int64, OwnedDeps, Uint64};

use crate::contract::{
    execute, instantiate, migrate, query, CONTRACT_NAME, CONTRACT_VERSION,
};
use crate::error::ContractError;
use crate::msg::{
    BlockParams, ConsensusParams, EvidenceParams, ExecuteMsg, InstantiateMsg,
    MigrateMsg, OwnerResponse, QueryMsg, ValidatorParams, VersionParams,
    MAX_MIN_BLOCK_COMMIT_DURATION_MS,
};
use crate::validate::MAX_BLOCK_SIZE_BYTES;

/// Owner address made by the default mock API.
const OWNER: &str = easy_addr::addr!("owner");
/// A different valid address with no rights.
const NOT_OWNER: &str = easy_addr::addr!("not-owner");

type Deps = OwnedDeps<MockStorage, MockApi, MockQuerier>;

/// Fixture shared with the Go package `x/xconsensus`.
const INSTANTIATE_FIXTURE: &str =
    include_str!("../testdata/instantiate_msg.json");
/// Fixture shared with the Go package `x/xconsensus`.
const UPDATE_PARAMS_FIXTURE: &str =
    include_str!("../testdata/update_params_msg.json");

/// Go `DefaultConsensusParams` (`bft/types/params.go`) with the commit
/// duration unset.
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
        version: VersionParams {
            app: Uint64::new(0),
        },
        min_block_commit_duration_ms: None,
    }
}

/// Instantiates the contract with `OWNER` and `params`.
fn setup(params: ConsensusParams) -> Deps {
    let mut deps = mock_dependencies();
    let msg = InstantiateMsg {
        owner: OWNER.to_string(),
        params,
    };
    instantiate(deps.as_mut(), mock_env(), mock_info(NOT_OWNER, &[]), msg)
        .expect("instantiate");
    deps
}

/// Queries the stored params.
fn query_params(deps: &Deps) -> ConsensusParams {
    from_json(query(deps.as_ref(), mock_env(), QueryMsg::Params {}).unwrap())
        .unwrap()
}

/// Queries the stored owner.
fn query_owner(deps: &Deps) -> String {
    let res: OwnerResponse =
        from_json(query(deps.as_ref(), mock_env(), QueryMsg::Owner {}).unwrap())
            .unwrap();
    res.owner
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

#[test]
fn instantiate_stores_owner_params_and_version() {
    let deps = setup(default_params());
    assert_eq!(query_params(&deps), default_params());
    // The owner comes from the message, not the instantiating sender.
    assert_eq!(query_owner(&deps), OWNER);
    let version = cw2::get_contract_version(deps.as_ref().storage).unwrap();
    assert_eq!(version.contract, CONTRACT_NAME);
    assert_eq!(version.version, CONTRACT_VERSION);
}

#[test]
fn instantiate_rejects_invalid_params_and_owner() {
    let mut deps = mock_dependencies();
    let mut params = default_params();
    params.block.max_bytes = Int64::new(0);
    let msg = InstantiateMsg {
        owner: OWNER.to_string(),
        params,
    };
    let err = instantiate(deps.as_mut(), mock_env(), mock_info(OWNER, &[]), msg)
        .unwrap_err();
    assert!(matches!(err, ContractError::InvalidParams(_)), "{err}");

    let msg = InstantiateMsg {
        owner: "Not An Address".to_string(),
        params: default_params(),
    };
    let err = instantiate(deps.as_mut(), mock_env(), mock_info(OWNER, &[]), msg)
        .unwrap_err();
    assert!(matches!(err, ContractError::Std(_)), "{err}");
}

#[test]
fn update_params_round_trip() {
    let mut deps = setup(default_params());
    let mut params = default_params();
    params.block.max_gas = Int64::new(150_000_000);
    params.validator.pub_key_types =
        vec!["ed25519".to_string(), "secp256k1".to_string()];
    params.version.app = Uint64::new(u64::MAX);
    params.min_block_commit_duration_ms = Some(200);
    update(&mut deps, OWNER, params.clone()).unwrap();
    assert_eq!(query_params(&deps), params);

    // Setting the value back to None unsets it.
    params.min_block_commit_duration_ms = None;
    update(&mut deps, OWNER, params.clone()).unwrap();
    assert_eq!(query_params(&deps).min_block_commit_duration_ms, None);
}

#[test]
fn update_params_rejects_invalid_and_keeps_old_value() {
    let mut deps = setup(default_params());
    let mut params = default_params();
    params.min_block_commit_duration_ms =
        Some(MAX_MIN_BLOCK_COMMIT_DURATION_MS + 1);
    let err = update(&mut deps, OWNER, params).unwrap_err();
    assert!(matches!(err, ContractError::InvalidParams(_)), "{err}");
    assert_eq!(query_params(&deps), default_params());
}

#[test]
fn owner_checks() {
    let mut deps = setup(default_params());
    let err = update(&mut deps, NOT_OWNER, default_params()).unwrap_err();
    assert_eq!(
        err,
        ContractError::Unauthorized {
            sender: NOT_OWNER.to_string(),
            owner: OWNER.to_string(),
        }
    );

    // The owner check runs before validation.
    let mut bad = default_params();
    bad.validator.pub_key_types = vec![];
    let err = update(&mut deps, NOT_OWNER, bad).unwrap_err();
    assert!(matches!(err, ContractError::Unauthorized { .. }), "{err}");

    let transfer = ExecuteMsg::UpdateOwner {
        owner: NOT_OWNER.to_string(),
    };
    let err = execute(
        deps.as_mut(),
        mock_env(),
        mock_info(NOT_OWNER, &[]),
        transfer.clone(),
    )
    .unwrap_err();
    assert!(matches!(err, ContractError::Unauthorized { .. }), "{err}");

    // An invalid new owner is rejected.
    let err = execute(
        deps.as_mut(),
        mock_env(),
        mock_info(OWNER, &[]),
        ExecuteMsg::UpdateOwner {
            owner: "Not An Address".to_string(),
        },
    )
    .unwrap_err();
    assert!(matches!(err, ContractError::Std(_)), "{err}");

    // Transfer, then the old owner loses rights and the new owner gains them.
    execute(deps.as_mut(), mock_env(), mock_info(OWNER, &[]), transfer).unwrap();
    assert_eq!(query_owner(&deps), NOT_OWNER);
    let err = update(&mut deps, OWNER, default_params()).unwrap_err();
    assert!(matches!(err, ContractError::Unauthorized { .. }), "{err}");
    update(&mut deps, NOT_OWNER, default_params()).unwrap();
}

#[test]
fn migrate_sets_version_and_keeps_state() {
    let mut deps = setup(default_params());
    cw2::set_contract_version(deps.as_mut().storage, CONTRACT_NAME, "0.0.0")
        .unwrap();
    migrate(deps.as_mut(), mock_env(), MigrateMsg {}).unwrap();
    let version = cw2::get_contract_version(deps.as_ref().storage).unwrap();
    assert_eq!(version.version, CONTRACT_VERSION);
    assert_eq!(query_params(&deps), default_params());
    assert_eq!(query_owner(&deps), OWNER);
}

/// One validation case: a mutation of the default params and the expected
/// result (`None` for valid, or a substring of the error).
struct Case {
    name: &'static str,
    mutate: fn(&mut ConsensusParams),
    err: Option<&'static str>,
}

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
        let via_exec = update(&mut deps, OWNER, params.clone());

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

/// The fixtures are the JSON contract with the Go package. They must decode
/// to the expected values and re-encode to the same JSON.
#[test]
fn json_fixtures_match_types() {
    let msg: InstantiateMsg = serde_json::from_str(INSTANTIATE_FIXTURE).unwrap();
    assert_eq!(msg.owner, "nibi1qqx5reauy4glpskmppy88pz25qp2py5yxvpxdt");
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
}

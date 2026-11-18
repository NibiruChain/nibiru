//! Typed access to the live x/sudo root. The native Wasm Stargate allowlist
//! returns JSON, so this module decodes the regenerated sudo response with
//! serde rather than using the protobuf-byte query helper.

use cosmwasm_std::{QuerierWrapper, StdResult};

use crate::proto::nibiru::sudo::{QuerySudoersRequest, QuerySudoersResponse};
use crate::proto::NibiruStargateQuery;

/// gRPC method declared in `proto/nibiru/sudo/v1/query.proto`. This method
/// retains the Query prefix: deriving it with the generic Cosmos query-name
/// convention would incorrectly produce `Query/Sudoers`.
pub const QUERY_SUDOERS_PATH: &str = "/nibiru.sudo.v1.Query/QuerySudoers";

/// Native zero-gas actor query, whose RPC name also retains the Query prefix.
pub const QUERY_ZERO_GAS_ACTORS_PATH: &str =
    "/nibiru.sudo.v1.Query/QueryZeroGasActors";

/// Query the native sudo module and return its current root address.
///
/// Generated serde implementations default omitted fields and tolerate extra
/// fields in native JSON responses. An absent root returns an empty string,
/// which callers must reject as having no authorized administrator. Transport
/// and decoding failures propagate instead of authorizing a cached owner.
pub fn query_sudo_root(querier: &QuerierWrapper) -> StdResult<String> {
    let request = QuerySudoersRequest {};
    let response: QuerySudoersResponse =
        querier.query(&request.into_stargate_query().map_err(|error| {
            cosmwasm_std::StdError::generic_err(error.to_string())
        })?)?;
    Ok(response.sudoers.map(|state| state.root).unwrap_or_default())
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::proto::nibiru::sudo::{RoleMembers, Sudoers};
    use cosmwasm_std::{
        from_json, Binary, ContractResult, Empty, Querier, QuerierResult,
        QueryRequest, SystemResult,
    };
    use prost::Message;

    /// The sudo service retains Query in its method names. Check both typed
    /// requests so generic Cosmos path derivation cannot silently break them.
    #[test]
    fn sudo_query_paths_match_native_service() {
        use crate::proto::nibiru::sudo::QueryZeroGasActorsRequest;
        assert_eq!(QuerySudoersRequest {}.path(), QUERY_SUDOERS_PATH);
        assert_eq!(
            QueryZeroGasActorsRequest {}.path(),
            QUERY_ZERO_GAS_ACTORS_PATH
        );
    }

    struct NativeSudoResponse(&'static str);
    impl Querier for NativeSudoResponse {
        fn raw_query(&self, bytes: &[u8]) -> QuerierResult {
            let request: QueryRequest<Empty> = from_json(bytes).unwrap();
            assert_eq!(
                request,
                QueryRequest::Stargate {
                    path: QUERY_SUDOERS_PATH.into(),
                    data: Binary::default(),
                }
            );
            SystemResult::Ok(ContractResult::Ok(Binary::from(self.0.as_bytes())))
        }
    }

    /// Regenerated state uses protobuf tag three for roles; the removed contracts
    /// field occupied tag two and must not be reused by the Rust representation.
    #[test]
    fn roles_use_current_proto_field_number() {
        let state = Sudoers {
            root: String::new(),
            roles: vec![RoleMembers {
                role: "wasm_deployer".into(),
                members: vec!["actor".into()],
            }],
        };
        let bytes = state.encode_to_vec();
        assert_eq!(bytes[0], 0x1a);
        assert_eq!(Sudoers::decode(bytes.as_slice()).unwrap(), state);
    }

    /// Native proto JSON may omit defaults and add fields. Neither case should
    /// break a root lookup or turn an absent root into a wildcard administrator.
    #[test]
    fn native_json_defaults_and_extra_fields_are_supported() {
        for (json, expected) in [
            (
                r#"{"sudoers":{"root":"live-root","roles":[{"role":"wasm_deployer","members":["actor"]}],"future_field":true},"future_response":1}"#,
                "live-root",
            ),
            (r#"{"sudoers":{}}"#, ""),
            (r#"{}"#, ""),
        ] {
            let response = NativeSudoResponse(json);
            assert_eq!(
                query_sudo_root(&QuerierWrapper::new(&response)).unwrap(),
                expected
            );
        }
    }

    /// Invalid module JSON is an authorization failure, not permission to use
    /// a previously observed root or assume the caller is privileged.
    #[test]
    fn malformed_native_json_is_rejected() {
        let response = NativeSudoResponse("invalid JSON");
        assert!(query_sudo_root(&QuerierWrapper::new(&response)).is_err());
    }
}

//! sudo.rs: Reads the x/sudo root, the only address allowed to change
//! x-consensus.

use cosmwasm_std::{Binary, QuerierWrapper, QueryRequest, StdResult};
use serde::{Deserialize, Serialize};

/// Stargate path of the x/sudo `QuerySudoers` gRPC method. It is on Nibiru's
/// Stargate accept list (`app/wasmext/stargate_query.go`), and the Wasm
/// keeper returns the response as proto JSON (`x/wasm/keeper/query_plugins.go`
/// `ConvertProtoToJSONMarshal`).
pub const QUERY_SUDOERS_PATH: &str = "/nibiru.sudo.v1.Query/QuerySudoers";

/// JSON form of `nibiru.sudo.v1.QuerySudoersResponse`, for example
/// `{"sudoers":{"root":"nibi1...","roles":[...]}}`.
///
/// Uses plain serde derives instead of `cw_serde`, so unknown fields such as
/// `roles` are ignored rather than rejected. This is the one place the
/// contract reads data it does not own, and a new x/sudo field must not
/// break every root check.
#[derive(Serialize, Deserialize, Clone, Debug, Default, PartialEq)]
pub struct QuerySudoersResponse {
    /// The x/sudo state.
    #[serde(default)]
    pub sudoers: Sudoers,
}

/// JSON form of `nibiru.sudo.v1.Sudoers`, reduced to the one field the
/// contract needs.
#[derive(Serialize, Deserialize, Clone, Debug, Default, PartialEq)]
pub struct Sudoers {
    /// Bech32 address of the x/sudo root. Empty when no root is set, which is
    /// the x/sudo genesis default. Defaults to empty when the JSON leaves it
    /// out, since proto JSON may omit empty strings.
    #[serde(default)]
    pub root: String,
}

/// Queries x/sudo for the current root address.
///
/// Returns an empty string when no root is set. Callers must treat that as
/// "nobody may change state", never as a wildcard.
pub fn query_sudo_root(querier: &QuerierWrapper) -> StdResult<String> {
    let res: QuerySudoersResponse = querier.query(&QueryRequest::Stargate {
        path: QUERY_SUDOERS_PATH.to_string(),
        // QuerySudoersRequest has no fields, so its proto encoding is
        // empty.
        data: Binary::default(),
    })?;
    Ok(res.sudoers.root)
}

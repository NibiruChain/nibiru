//! scripts/proto_clean.rs:
//!
//! Run with: `cargo run --bin proto_clean`
//!
//! ## Procedure
//!
//! 1. Walk through all the files in the nibiru-std/src/proto directory.
//! 2. For each file, read its content and identify lines that import types with
//!    multiple super components.
//! 3. Classify each import based on the first non-super part, then replace the
//!    super components with crate::proto::cosmo or crate::proto::tendemint based
//!    on the classification.
//! 4. Write the modified content back to each file.

pub static PROTO_PATH: &str = "../nibiru-std/src/proto/buf";

pub fn main() {
    println!("Running proto_clean.rs...");

    // Walk through each file in the directory
    for entry in walkdir::WalkDir::new(PROTO_PATH)
        .into_iter()
        .filter_map(|e| e.ok())
        // filter out directories
        .filter(|e| !e.file_type().is_dir())
        .filter(|e| {
            let fname = e.file_name().to_string_lossy();
            fname.starts_with("cosmos")
                || fname.starts_with("nibiru")
                || fname.starts_with("eth")
        })
    {
        // Get the path of the file we're going to clean.
        let path_to_clean = entry.path().to_string_lossy();
        println!("cleaning {}", path_to_clean);

        // Read the file content and replace contents
        clean_file_imports_inplace(&path_to_clean).unwrap_or_else(|_| {
            panic!("failed to clean proto file: {}", path_to_clean)
        })
    }

    println!("ran proto_clean.rs successfully");
}

pub fn clean_file_imports(
    rust_proto_path: &str,
) -> Result<String, Box<dyn std::error::Error>> {
    // Define a regular expression to match `super::` imports
    let re = regex::Regex::new(r"super::(?:super::)+[\w:]+")?;
    let content = std::fs::read_to_string(rust_proto_path)?;
    // Replace all matches in the file content using the provided function
    let updated_content = re.replace_all(&content, |caps: &regex::Captures| {
        let matched = &caps[0]; // Get the entire matched string
        match super_import_to_clean(matched) {
            Ok(cleaned) => cleaned,
            Err(err) => {
                eprintln!("Error cleaning import '{}': {:?}", matched, err);
                matched.to_string() // If there's an error, leave the import unchanged
            }
        }
    });

    Ok(updated_content.to_string())
}

/// Runs clean_file_imports and writes new contents back to the input file.
pub fn clean_file_imports_inplace(
    rust_proto_path: &str,
) -> Result<(), Box<dyn std::error::Error>> {
    match clean_file_imports(rust_proto_path) {
        Ok(content) => {
            std::fs::write(rust_proto_path, content)?;
            Ok(())
        }
        Err(err) => Err(err),
    }
}

#[derive(Debug)]
pub struct CustomError(String);

impl std::fmt::Display for CustomError {
    fn fmt(&self, f: &mut std::fmt::Formatter) -> std::fmt::Result {
        write!(f, "{}", self.0)
    }
}

impl std::error::Error for CustomError {}

/// Cleans a "super import" string and rebuilds it with the proper prefix.
pub fn super_import_to_clean(
    super_import: &str,
) -> Result<String, Box<CustomError>> {
    // Split import string into separate elements
    let elems: Vec<&str> = super_import.split("::").collect();

    let submodule_idx = elems
        .iter()
        .position(|&elem| !elem.is_empty() && elem != "super")
        .ok_or_else(|| {
            Box::new(CustomError(format!(
                "Only super elements found in import '{}'",
                super_import
            )))
        })?;

    let elem = elems[submodule_idx];
    let prefix: &str;
    if proto_submodules::is_submod_cosmos(elem) {
        prefix = "crate::proto::cosmos"
    } else if proto_submodules::is_mod_tendermint(elem)
        || proto_submodules::is_mod_cosmos(elem)
        || proto_submodules::is_mod_eth(elem)
    {
        prefix = "crate::proto"
    } else if proto_submodules::is_submod_cosmos_base(elem) {
        prefix = "crate::proto::cosmos::base"
    } else if proto_submodules::is_submod_cosmos_tx(elem) {
        prefix = "crate::proto::cosmos::tx"
    } else if proto_submodules::is_submod_cosmos_crypto(elem) {
        prefix = "crate::proto::cosmos::crypto"
    } else if proto_submodules::is_submod_eth(elem) {
        prefix = "crate::proto::eth"
    } else {
        return Err(Box::new(CustomError(format!(
            "Unrecognized import submodule: {}",
            super_import
        ))));
    };

    let out_str = format!("{}::{}", prefix, elems[submodule_idx..].join("::"));
    Ok(out_str)
}

mod proto_submodules {

    pub fn is_submod_cosmos(s: &str) -> bool {
        COSMOS.contains(&s)
    }

    pub fn is_mod_tendermint(s: &str) -> bool {
        TENDERMINT.contains(&s)
    }

    pub fn is_submod_cosmos_base(s: &str) -> bool {
        matches!(s, "query")
    }

    pub fn is_submod_cosmos_tx(s: &str) -> bool {
        matches!(s, "signing")
    }

    pub fn is_mod_cosmos(s: &str) -> bool {
        matches!(s, "cosmos")
    }

    pub fn is_mod_eth(s: &str) -> bool {
        matches!(s, "eth")
    }

    /// List of all proto package names beginning with "cosmos". This list only
    /// contains immediate children.
    pub static COSMOS: [&str; 25] = [
        "base",
        "bank",
        "distribution",
        "app",
        "auth",
        "authz",
        "autocli",
        "capability",
        "consensus",
        "crypto",
        "evidence",
        "feegrant",
        "genutil",
        "gov",
        "group",
        "mint",
        "nft",
        "orm",
        "params",
        "reflection",
        "slashing",
        "staking",
        "tx",
        "upgrade",
        "vesting",
    ];

    /// COSMOS_CRYPTO: Names of the proto packages: cosmos.crypto.*
    pub static COSMOS_CRYPTO: [&str; 6] = [
        "ed25519",
        "hd",
        "keyring",
        "multisig",
        "secp256k1",
        "secp256r1",
    ];

    pub fn is_submod_cosmos_crypto(s: &str) -> bool {
        COSMOS_CRYPTO.contains(&s)
    }

    pub static TENDERMINT: [&str; 1] = ["tendermint"];

    /// List of all proto package names beginning with "eth". This list only
    /// contains immediate children.
    pub static ETH: [&str; 2] = ["evm", "types"];

    /// True if the proto package name is a an immediate child of "eth".
    /// For example, the existence of "eth.evm.v1.rs" implies that "evm" will
    /// return true.
    pub fn is_submod_eth(s: &str) -> bool {
        ETH.contains(&s)
    }
}

#[cfg(test)]
mod tests {
    use std::fs;

    #[derive(Debug, PartialEq)]
    pub struct TestCase {
        input: &'static str,
        want_err: bool,
        want_out: Option<&'static str>,
    }

    #[test]
    fn test_super_import_to_clean() {
        let test_cases: Vec<TestCase> = vec![
            TestCase {
                input: "::super::super::super::bank::foo",
                want_err: false,
                want_out: Some("crate::proto::cosmos::bank::foo"),
            },
            TestCase {
                input: "::super::tendermint::xyz",
                want_err: false,
                want_out: Some("crate::proto::tendermint::xyz"),
            },
            TestCase {
                input: "abcxyz",
                want_err: true,
                want_out: None,
            },
            TestCase {
                input: "super::super::super::hd::v1::Bip69Params",
                want_err: false,
                want_out: Some(
                    "crate::proto::cosmos::crypto::hd::v1::Bip69Params",
                ),
            },
            TestCase {
                input: "super::super::cosmos::base::v1beta1::Coin",
                want_err: false,
                want_out: Some("crate::proto::cosmos::base::v1beta1::Coin"),
            },
            TestCase {
                input: "::super::super::eth::evm::v1::FunToken",
                want_err: false,
                want_out: Some("crate::proto::eth::evm::v1::FunToken"),
            },
        ];

        for (i, test_case) in test_cases.iter().enumerate() {
            let result = super::super_import_to_clean(test_case.input);

            // Check if the result is an error as expected
            if test_case.want_err {
                assert!(result.is_err(), "Test case {} failed", i);
            } else {
                assert!(result.is_ok(), "Test case {} failed", i);
                // Check the expected output
                if let Ok(cleaned) = result {
                    assert_eq!(
                        cleaned,
                        test_case.want_out.unwrap(),
                        "Test case {} failed",
                        i
                    );
                }
            }
        }
    }

    #[test]
    fn fixture_proto_clean() {
        let dirty_path = "test/fixture_proto_dirty.rs";
        let result = super::clean_file_imports(dirty_path);
        assert!(result.is_ok());

        let clean_path = "test/fixture_proto_clean.rs";
        let want_result = fs::read_to_string(clean_path);
        assert!(want_result.is_ok());

        assert_eq!(result.unwrap(), want_result.unwrap());
    }
}

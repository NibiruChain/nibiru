package wasmvm

// ExpectedVersion is the version of the in-tree libwasmvm Rust crate.
//
// Keep this value in sync with libwasmvm/Cargo.toml. It is used by nibid at
// startup to ensure that a dynamically linked libwasmvm matches the version
// the binary was built against.
const ExpectedVersion = "1.5.9"

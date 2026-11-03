//! Root-controlled configuration and self-migration use the shared typed
//! x/sudo query. Scheduling and cancellation instead require the native
//! governance module account; the CosmWasm sudo entry point trusts the host.
//!
//! nibiru_std::sudo owns QUERY_SUDOERS_PATH and decodes the regenerated native
//! response types, including roles and omitted default values.

pub use nibiru_std::sudo::query_sudo_root;

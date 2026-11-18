//! Canonical upgrade state behind Nibiru's native module API.
//!
//! Governance schedules/cancels through execute. The live x/sudo root manages
//! configuration and self-migration. Native lifecycle calls use CosmWasm sudo.
pub mod contract;
pub mod msg;
pub mod state;
pub mod sudo;

#[cfg(test)]
mod tests;

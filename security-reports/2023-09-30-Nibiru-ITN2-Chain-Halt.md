# Postmortem

## Summary

On Sept 30 @ 22:16:01 UTC time, the `nibiru-itn-2` network halted at block `1131575`. 

## Root Cause

The binary was missing a wasm extension that copies the `wasm` smart contract folder for state syncs. Nodes that joined the network via state sync were missing `wasm` smart contracts. Later on, these nodes became validator nodes, and the set of smart contracts differed between validator nodes.

A tx in block `1131574` was submitted against one of these missing smart contracts (with code_id 3). Some validators were able to execute the tx successfully and other validators errored out since they didn’t have the wasm smart contract in their local disk. Hence the chain halted while validating the `app_hash` in block `1131575`. 

## Resolution

The issue was fixed in [PR #1616](https://github.com/NibiruChain/nibiru/pull/1616) and backported to the [v0.21.x release branch](https://github.com/NibiruChain/nibiru/tree/releases/v0.21.x) (currently in [v0.21.11](https://github.com/NibiruChain/nibiru/releases/tag/v0.21.11)).
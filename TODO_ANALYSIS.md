# TODO/NOTE Analysis: Work Stream Prioritization

## Executive Summary

This document categorizes 100+ TODOs, NOTEs, FIXMEs, and related items from the codebase into work streams suitable for engineers looking to make meaningful impact without getting stuck on complex, multi-week projects.

---

## 🔴 High Priority - Security & Critical Bugs

**Impact:** High - Security vulnerabilities, data integrity issues, or production bugs  
**Complexity:** Low to Medium  
**Estimated Effort:** 1-5 days each

### Security Issues
- [ ] **`app/ante.go:64`** - `TODO: spike(security): Does minimum gas price of 0 pose a risk?`  
   - Linked issue: #1916
   - **Action:** Security assessment of zero gas price implications
   - **Impact:** Could prevent gas-related attacks

- [ ] **`eth/rpc/rpcapi/websockets.go:90`** - `FIXME: this shouldn't be hardcoded to localhost`  
   - **Action:** Make RPC address configurable instead of hardcoded
   - **Impact:** Configuration/security issue, affects deployment flexibility
   - **Complexity:** Low - likely 1-2 days

- [ ] **`app/ante.go:53`** - `TODO: bug(security): Authz is unsafe. Let's include a guard to make things safer.`  
   - Linked issue: #1915
   - **Note:** `AnteDecAuthzGuard` already exists - may need enhancement or documentation
   - **Action:** Verify guard is sufficient or enhance it
   - **Impact:** Prevents unauthorized access via authz module

### Error Handling
- [ ] **`x/bank/keeper/keeper.go:543`** - `TODO: return error on account.TrackDelegation`  
- [ ] **`x/bank/keeper/keeper.go:560`** - `TODO: return error on account.TrackUndelegation`  
   - **Action:** Add error handling for vesting account operations
   - **Impact:** Better error propagation and debugging
   - **Complexity:** Low - 1-2 days total

---

## 🟡 Medium Priority - Feature Implementation

**Impact:** Medium-High - Adds functionality or improves user experience  
**Complexity:** Medium (good scope for meaningful work)  
**Estimated Effort:** 3-10 days each

### EVM RPC Features
- [ ] **`evm-e2e/test/debug_queries.test.ts:89`** - `TODO: feat(evm-rpc): impl the debug_getBadBlocks EVM RPC method`  
- [ ] **`evm-e2e/test/debug_queries.test.ts:102`** - `TODO: feat(evm-rpc): impl the debug_storageRangeAt EVM RPC method`  
   - **Action:** Implement debug RPC methods for EVM compatibility
   - **Impact:** Improves developer tooling and debugging capabilities
   - **Complexity:** Medium - 3-5 days each

- [ ] **`eth/rpc/rpcapi/backend.go:60`** - `TODO: feat(eth): Implement the cosmos JSON-RPC defined by Wallet Connect V2`  
   - **Action:** Implement Wallet Connect V2 support
   - **Impact:** Enables wallet connectivity
   - **Complexity:** Medium-High - 5-10 days

- [ ] **`eth/rpc/rpcapi/chain_info.go:188`** - `TODO: feat(eth): dynamic fees`  
   - **Action:** Implement EIP-1559 dynamic fee support
   - **Impact:** Modern fee mechanism for EVM transactions
   - **Complexity:** Medium-High - 5-8 days

- [ ] **`eth/rpc/rpcapi/blocks.go:481`** - `TODO: feat(evm-backend): Add tx receipts in gethcore.NewBlock`  
- [ ] **`eth/rpc/rpcapi/blocks.go:485`** - `TODO: feat: See if we can simulate Trie behavior on CometBFT`  
   - **Action:** Enhance block data for EVM compatibility
   - **Impact:** Better compatibility with Ethereum tooling
   - **Complexity:** Medium - 3-5 days each

### Precompile Contracts
- [ ] **`x/evm/precompile/precompile.go:63`** - `TODO: feat(evm): implement precompiled contracts for ibc transfer`  
- [ ] **`x/evm/precompile/precompile.go:66`** - `TODO: feat(evm): implement precompiled contracts for staking`  
   - **Action:** Add precompiled contracts for IBC and staking operations
   - **Impact:** Enables EVM contracts to interact with Cosmos SDK modules
   - **Complexity:** Medium-High - 5-10 days each

- [ ] **`x/evm/precompile/funtoken_test.go:310`** - `TODO: [feat] Handle WNIBI as a case in the FunToken precompile`  
   - **Action:** Add WNIBI handling to FunToken precompile
   - **Impact:** Better token handling for wrapped native token
   - **Complexity:** Low-Medium - 2-4 days

### Configuration & Flags
- [ ] **`app/app.go:440`** - `TODO: feat(evm): enable app/server/config flag for Evm MaxTxGasWanted`  
   - **Action:** Add config flag for EVM max transaction gas
   - **Impact:** Operational flexibility
   - **Complexity:** Low-Medium - 2-3 days

---

## 🟢 Medium-Low Priority - Code Quality & Refactoring

**Impact:** Medium - Improves maintainability and reduces technical debt  
**Complexity:** Low to Medium  
**Estimated Effort:** 1-5 days each

### Code Organization
- [ ] **`x/nutil/address.go:26`** - `TODO: (realu) Move to collections library`  
   - **Action:** Refactor address utilities to use collections library
   - **Impact:** Better code organization
   - **Complexity:** Low - 1-2 days

- [ ] **`eth/rpc/rpcapi/api_eth.go:21`** - `TODO: Remove this interface over since it's largely unused and the expected`  
   - **Action:** Remove unused interface
   - **Impact:** Code cleanup
   - **Complexity:** Low - 1 day

- [x] **`x/nutil/testutil/testnetwork/network_config.go:21`** - `TODO: Remove!`  
   - **Action:** Remove deprecated code
   - **Impact:** Code cleanup
   - **Complexity:** Low - 1 day

### Testing Improvements
- [x] **`x/bank/keeper/invariants.go:79`** - `TODO: test TotalSupply`  
   - **Action:** Add invariant test for total supply
   - **Impact:** Better test coverage
   - **Complexity:** Low - 1-2 days

- [x] **`x/bank/keeper/nibiru_ext_test.go:437`** - `TODO: test -> Create mismatch on purpose by burning to create sad path`  
   - **Action:** Add negative test case
   - **Impact:** Better test coverage
   - **Complexity:** Low - 1 day

- [ ] **`eth/rpc/rpcapi/api_eth_test.go:266,277,286`** - Multiple `TODO: add more checks`  
   - **Action:** Enhance test assertions
   - **Impact:** Better test coverage
   - **Complexity:** Low - 1-2 days total

- [ ] **`x/evm/tx_data_dynamic_fee_test.go:130`** - `TODO: Test for different pointers`  
- [ ] **`x/evm/tx_data_legacy_test.go:55`** - `TODO: Test for different pointers`  
   - **Action:** Add pointer equality tests
   - **Impact:** Better test coverage
   - **Complexity:** Low - 1 day each

### Documentation
- [ ] **`eth/rpc/types.go:4`** - `TODO: docs(eth-rpc): Explain types further`  
   - **Action:** Add documentation for RPC types
   - **Impact:** Better developer experience
   - **Complexity:** Low - 1-2 days

- [ ] **`eth/crypto/hd/algorithm.go:93`** - `TODO: add links to godocs of the two methods or implementations of them`  
   - **Action:** Add documentation links
   - **Impact:** Better code documentation
   - **Complexity:** Low - 1 day

---

## 🔵 Low Priority - Cleanup & Technical Debt

**Impact:** Low-Medium - Code hygiene and future-proofing  
**Complexity:** Low  
**Estimated Effort:** 1-3 days each

### Legacy Code Removal
- [x] **`x/devgas/v1/types/params_legacy.go:3`** - `TODO: Remove this and params_legacy_test.go after v0.47.x (v16) upgrade`  
   - **Action:** Remove legacy params after upgrade completes
   - **Impact:** Code cleanup
   - **Complexity:** Low - 1 day (wait for upgrade)

- [ ] **`eth/crypto/ethsecp256k1/ethsecp256k1.go:121`** - `TODO: remove`  
   - **Action:** Remove deprecated code
   - **Impact:** Code cleanup
   - **Complexity:** Low - 1 day (verify no dependencies)

### Minor Improvements
- [ ] **`gosdk/broadcast.go:142`** - `TODO: implement`  
   - **Action:** Complete implementation (context needed)
   - **Impact:** Depends on what's missing
   - **Complexity:** Unknown - needs investigation

- [ ] **`eth/gas_limit.go:102`** - `TODO: Should we set the consumed field after overflow checking?`  
   - **Action:** Code review/decision needed
   - **Impact:** Code correctness
   - **Complexity:** Low - 1 day

- [ ] **`x/evm/evmante/evmante_increment_sender_seq_test.go:93`** - `TODO: Is there a better strategy than panicking here?`  
   - **Action:** Evaluate error handling strategy
   - **Impact:** Code quality
   - **Complexity:** Low-Medium - 1-2 days

- [ ] **`x/nutil/testutil/testnetwork/network.go:647`** - `TODO: Is there a cleaner way to do this with a guaranteed synchronous on`  
   - **Action:** Refactor for cleaner async handling
   - **Impact:** Code quality
   - **Complexity:** Low-Medium - 1-2 days

- [ ] **`x/bank/keeper/view.go:149`** - `TODO: revisit, for now, panic here to keep same behavior as in 0.42`  
   - **Action:** Review panic behavior, consider error handling
   - **Impact:** Code quality
   - **Complexity:** Low - 1 day

- [ ] **`app/sim/config.go:23`** - `TODO: Remove in favor of binary search for invariant violation`  
   - **Action:** Implement binary search for invariant detection
   - **Impact:** Performance improvement
   - **Complexity:** Low-Medium - 2-3 days

- [ ] **`x/evm/evmstate/genesis.go:91`** - `TODO: find the way to get eth contract addresses from the evm keeper`  
   - **Action:** Implement proper address retrieval
   - **Impact:** Code correctness
   - **Complexity:** Low-Medium - 1-2 days

- [ ] **`x/evm/vmtracer.go:31`** - `TODO: feat(evm-vmtracer): enable additional log configuration`  
   - **Action:** Add configurable logging for VM tracer
   - **Impact:** Developer experience
   - **Complexity:** Low-Medium - 2-3 days

- [ ] **`x/evm/evmstate/keeper.go:113`** - `TODO: (someday maybe): Consider making base fee dynamic based on`  
   - **Action:** Design dynamic fee mechanism
   - **Impact:** Future enhancement
   - **Complexity:** Medium - 3-5 days

- [ ] **`x/evm/evmstate/call_contract.go:44`** - `TODO: UD-DEBUG: CallContract - Remove commit`  
   - **Action:** Debug and remove commit if unnecessary
   - **Impact:** Code correctness
   - **Complexity:** Low-Medium - 1-2 days

- [ ] **`eth/rpc/rpcapi/nonce_test.go:68`** - `TODO: perf(evmante): Make per-block uncommitted txs execute in a batch`  
   - **Action:** Optimize transaction batching
   - **Impact:** Performance improvement
   - **Complexity:** Medium - 3-5 days

- [ ] **`eth/rpc/rpcapi/websockets.go:373`** - `TODO: handle extra params`  
- [ ] **`eth/rpc/rpcapi/websockets.go:395`** - `TODO: use events`  
   - **Action:** Enhance websocket handling
   - **Impact:** Feature completeness
   - **Complexity:** Low-Medium - 2-3 days each

- [ ] **`eth/rpc/rpcapi/api_eth_test.go:323`** - `TODO: the backend method is stubbed to 0`  
   - **Action:** Implement stubbed method
   - **Impact:** Feature completeness
   - **Complexity:** Low-Medium - 1-2 days

- [ ] **`eth/rpc/rpcapi/account_info.go:129`** - `NOTE: The StorageHash is blank. Consider whether this is useful`  
   - **Action:** Evaluate and implement if needed
   - **Impact:** Feature completeness
   - **Complexity:** Low - 1 day

- [ ] **`x/bank/testutil/helpers.go:14,28`** - Multiple `TODO: Instead of using the mint module account`  
   - **Action:** Refactor test helpers
   - **Impact:** Code quality
   - **Complexity:** Low - 1-2 days

- [ ] **`app/ante/testutil_test.go:113`** - `TODO: UD-DEBUG: REMOVED: Don't create new accounts here - use the existing ones`  
   - **Action:** Refactor test to use existing accounts
   - **Impact:** Test quality
   - **Complexity:** Low - 1 day

- [ ] **`app/server/util.go:54`** - `TODO: check indexer tx command`  
   - **Action:** Review indexer command
   - **Impact:** Code quality
   - **Complexity:** Low - 1 day

- [ ] **`x/epochs/simulation/genesis.go:53`** - `TODO: Do some randomization later`  
   - **Action:** Add randomization to simulation
   - **Impact:** Test quality
   - **Complexity:** Low - 1 day

- [ ] **`eth/crypto/hd/algorithm.go:107`** - `TODO: modulo P`  
   - **Action:** Add modulo operation (context needed)
   - **Impact:** Code correctness
   - **Complexity:** Low - 1 day

- [ ] **`eth/rpc/rpcapi/api_net.go:17`** - `TODO: epic: test(eth-rpc): "github.com/NibiruChain/nibiru/v2/x/common/testutil/cli"`  
   - **Action:** Add integration tests
   - **Impact:** Test coverage
   - **Complexity:** Medium - 3-5 days

---

## 📝 Informational Notes (No Action Required)

These are documentation/contextual notes that don't require implementation work:

- Multiple `NOTE: All parameters must be supplied` in proto files
- `NOTE: This is required for the GetSignBytes function` in codec files
- `NOTE: This is used solely for migration of x/params managed parameters`
- `NOTE: address, topics and data are consensus fields` in EVM proto files
- `NOTE: Using the verbose flag breaks the coverage reporting in CI`
- `NOTE: The brackets around the word "Unreleased" are required to pass the CI test`
- `NOTE: unsafe → assumes pre-validated inputs`
- Various other informational notes explaining code behavior

---

## 🎯 Recommended Work Streams (Prioritized)

### Stream 1: Security & Error Handling (High Impact, Low Complexity)
**Duration:** 1-2 weeks  
**Items:**
- [ ] Fix hardcoded localhost in websockets (2 days)
- [ ] Add error handling for TrackDelegation/TrackUndelegation (2 days)
- [ ] Security assessment for zero gas price (2-3 days)
- [ ] Review/verify Authz guard (1-2 days)

**Why:** Quick wins with high security/correctness impact

### Stream 2: EVM RPC Debug Methods (Medium Impact, Medium Complexity)
**Duration:** 1-2 weeks  
**Items:**
- [ ] Implement `debug_getBadBlocks` (3-5 days)
- [ ] Implement `debug_storageRangeAt` (3-5 days)

**Why:** Adds developer tooling, well-scoped features

### Stream 3: Testing & Test Coverage (Medium Impact, Low Complexity)
**Duration:** 1 week  
**Items:**
- [ ] Test TotalSupply invariant (1-2 days)
- [ ] Add negative test cases (1-2 days)
- [ ] Enhance RPC test assertions (2-3 days)
- [ ] Add pointer equality tests (1-2 days)

**Why:** Quick improvements to code quality and reliability

### Stream 4: Code Cleanup & Refactoring (Low-Medium Impact, Low Complexity)
**Duration:** 1 week  
**Items:**
- [ ] Remove unused interfaces (1 day)
- [ ] Move address utils to collections (1-2 days)
- [ ] Remove deprecated code (1-2 days)
- [ ] Add documentation (2-3 days)

**Why:** Reduces technical debt, improves maintainability

### Stream 5: Precompile Contracts (High Impact, Medium-High Complexity)
**Duration:** 2-3 weeks  
**Items:**
- [ ] Implement IBC transfer precompile (5-10 days)
- [ ] Implement staking precompile (5-10 days)
- [ ] Handle WNIBI in FunToken (2-4 days)

**Why:** Significant feature addition, but more complex

---

## 📊 Summary Statistics

- **Total Items:** ~100
- **High Priority (Security/Bugs):** 5 items
- **Medium Priority (Features):** 10 items
- **Medium-Low Priority (Quality):** 9 items
- **Low Priority (Cleanup):** 23 items
- **Informational Notes:** ~50+ items (no action needed)

---

## 💡 Recommendations

1. **Start with Stream 1** - Security and error handling provide immediate value with low risk
2. **Stream 2 or 3** - Good for building confidence and making visible progress
3. **Stream 4** - Good for learning the codebase while making improvements
4. **Stream 5** - Reserve for more experienced engineers or when ready for larger features

Most items are well-scoped and can be completed independently, making them ideal for parallel work or incremental progress.


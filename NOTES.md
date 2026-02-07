# Notes: EVM Ante Handler for Zero-Balance Gas Transactions

- [Question Posed](#question-posed)
- [Answer Summary](#answer-summary)
- [Relevant Code Locations](#relevant-code-locations)
  - [Entry Point: Ante Handler Routing](#entry-point-ante-handler-routing)
  - [EVM Ante Step Order](#evm-ante-step-order)
- [Checks That Need a Custom Bypass](#checks-that-need-a-custom-bypass)
  - [1. AnteStepVerifyEthAcc — Balance vs. Tx Cost](#1-antestepverifyethacc--balance-vs-tx-cost)
  - [2. AnteStepCanTransfer — Gas Cap and Value Transfer](#2-antestepcantransfer--gas-cap-and-value-transfer)
  - [3. AnteStepDeductGas — Fee Deduction](#3-antestepdeductgas--fee-deduction)
  - [4. AnteStepMempoolGasPrice (Optional)](#4-antestepmempoolgasprice-optional)
- [Reference: Non-EVM Zero-Gas Pattern](#reference-non-evm-zero-gas-pattern)
- [Conceptual Model: Gas Payment Flow](#conceptual-model-gas-payment-flow)
- [Chosen Implementation: Conditional Bypass](#chosen-implementation-conditional-bypass)
- [How It Fits the Current Design](#how-it-fits-the-current-design)
  - [Gas Payment Phases for Zero-Gas](#gas-payment-phases-for-zero-gas)
  - [Ante to Msg_server Communication](#ante-to-msg_server-communication)
- [Implementation Plan](#implementation-plan)
  - [Edge Cases](#edge-cases)
- [Alternative Approach: Credit to Undo (Not Implemented)](#alternative-approach-credit-to-undo-not-implemented)
- [Implementation Tracker](#implementation-tracker)
  - [Design invariants and execution model](#design-invariants-and-execution-model)
  - [Ante chain: detection and skip behavior](#ante-chain-detection-and-skip-behavior)
  - [Msg handler and testing](#msg-handler-and-testing)
  - [Test-to-Claim Mapping](#test-to-claim-mapping)
  - [Test Coverage Gaps](#test-coverage-gaps)
  - [Files and flow](#files-and-flow)
- [Design Clarifications](#design-clarifications)

Date: 2026-02-07

## Question Posed

Suppose we wanted to implement an EVM ante handler that allowed a **specific type
of transaction** to run **without requiring a balance of gas**—what code would
that handler be related to? Which checks in the current ante stack would be
relevant and need some custom bypass?

---

## Answer Summary

An EVM "zero-balance gas" handler would live in the **EVM ante stack**
(`x/evm/evmante/`). The EVM path is completely separate from the non-EVM (Cosmos
SDK) ante chain. Several existing steps enforce gas/balance requirements and
would need conditional bypasses for the special transaction type.

---

## Relevant Code Locations

### Entry Point: Ante Handler Routing

EVM vs non-EVM transactions are routed in `app/ante.go`:

```go
// app/ante.go, lines 31-36
if !evm.IsEthTx(tx) {
    anteHandler = NewAnteHandlerNonEVM(keepers.PublicKeepers, options)
    return anteHandler(ctx, tx, sim)
}
anteHandler = evmante.NewAnteHandlerEvm(options)
return anteHandler(ctx, tx, sim)
```

- **File:** `app/ante.go`
- **Lines:** 31–36

### EVM Ante Step Order

The EVM ante handler is defined in `x/evm/evmante/all_evmante.go`:

```go
// x/evm/evmante/all_evmante.go (see current order in repo)
steps := []AnteStep{
    AnteStepSetupCtx,                    // outermost
    EthSigVerification,
    AnteStepValidateBasic,
    AnteStepDetectZeroGas,               // ← detect zero-gas tx, set context marker (before steps that skip)
    AnteStepMempoolGasPrice,             // ← mempool min gas price (CheckTx); skips if zero-gas
    AnteStepBlockGasMeter,
    AnteStepVerifyEthAcc,                // ← validation + account creation always; skip only balance vs. cost if zero-gas
    AnteStepCanTransfer,                 // ← gas cap, value transfer; runs always (value no-ops when 0)
    AnteStepGasWanted,
    AnteStepDeductGas,                   // ← deduct fees from sender; skips if zero-gas
    AnteStepIncrementNonce,
    AnteStepEmitPendingEvent,
    AnteStepFiniteGasLimitForABCIDeliverTx,
}
```

- **File:** `x/evm/evmante/all_evmante.go`

---

## Checks That Need a Custom Bypass

### 1. AnteStepVerifyEthAcc — Balance vs. Tx Cost

Validates from address, ensures sender account exists (creates if missing), and checks that the sender’s balance covers the full transaction cost (fees + value).

| Location  | File                                 | Lines  |
|-----------|--------------------------------------|--------|
| Ante step | `x/evm/evmante/evmante_can_transfer.go` | 18–76  |
| Core check| `x/evm/evmstate/gas_fees.go`          | 97–117 |

**Key logic in `evmstate.CheckSenderBalance`:** `cost := txData.Cost(); balanceWei.ToBig().Cmp(cost) < 0` → insufficient funds.

**Bypass:** For zero-gas, **only** skip the balance-vs-cost check. From validation and account creation still run (required for first-time onboarding and nonce consistency—see [Account verification vs. gas bypass](#account-verification-vs-gas-bypass-functional-requirement)).

---

### 2. AnteStepCanTransfer — Gas Cap and Value Transfer

Validates:

1. `gasFeeCap >= baseFee`
2. If `value > 0`, balance >= value

| Location  | File                                 | Lines   |
|-----------|--------------------------------------|---------|
| Ante step | `x/evm/evmante/evmante_can_transfer.go` | 82–138  |

**Key checks:**

- Gas fee cap vs base fee: lines 110–116
- Value transfer vs balance: lines 121–132

**Behavior:** CanTransfer runs for zero-gas txs. Gas cap check passes (EffectiveGasFeeCapWei returns max(baseFee, txCap)). Value check no-ops when `tx.Value == 0` (required by eligibility). No bypass; structural validity is enforced.

---

### 3. AnteStepDeductGas — Fee Deduction

Deducts transaction fees from the sender and sends them to the fee collector. For zero-gas txs, the entire step is skipped before fee computation.

| Location     | File                                 | Lines    |
|--------------|--------------------------------------|----------|
| Ante step    | `x/evm/evmante/evmante_gas_consume.go` | 99–159   |
| Fee calc     | `x/evm/evmstate/gas_fees.go`          | 159–212 (VerifyFee) |
| Deduction    | `x/evm/evmstate/gas_fees.go`          | 119–138 (DeductTxCostsFromUserBalance) |

**Primary bypass for zero-gas (runs before VerifyFee):**

```go
// evmante_gas_consume.go, lines 104-106
if evm.IsZeroGasEthTx(sdb.Ctx()) {
    return nil
}
```

A secondary path skips deduction when `fees.IsZero()` (after VerifyFee); zero-gas txs never reach it.

---

### 4. AnteStepMempoolGasPrice (Optional)

Enforces minimum gas price during CheckTx for mempool admission.

| Location  | File                                 | Lines   |
|-----------|--------------------------------------|---------|
| Ante step | `x/evm/evmante/evmante_mempool_fees.go` | 38–75   |

Relevant if the special transactions use zero or very low gas price. Skip this step when the tx is identified as the special type.

---

## Reference: Non-EVM Zero-Gas Pattern

The non-EVM path has an existing zero-gas pattern for Wasm `MsgExecuteContract`:

| Component             | File                 | Purpose                                                |
|-----------------------|----------------------|--------------------------------------------------------|
| AnteDecZeroGasActors  | `app/ante/fixed_gas.go` | Sets fixed zero-gas meter for whitelisted sender+contract |
| DeductFeeDecorator    | `app/ante/deduct_fee.go` | Skips fee deduction when `isZeroGasMeter(ctx)` is true  |
| ZeroGasActors state   | `x/sudo/`            | Governance-managed whitelist (senders + contracts)     |

- `app/ante/fixed_gas.go`, lines 71–109
- `app/ante/deduct_fee.go`, lines 53–57, 159–169 (`isZeroGasMeter`)

EVM does not use this path; it has its own ante stack, so an analogous mechanism must be implemented there.

---

## Conceptual Model: Gas Payment Flow

At a high level, gas payment for an EVM transaction works as follows:

1. **Arrival** — The transaction shows up.
2. **Check** — The signer/sender's balance is verified to cover the transaction cost.
3. **Deduct** — The full upfront cost (gas limit × gas price) is taken from the sender and sent to the fee collector.
4. **Execute** — The EVM runs the transaction.
5. **Refund** — Unused gas is refunded: some funds move from the fee collector back to the sender.

So a user's gas balance **G** ends up in two places:
- A portion goes to the **fee collector** (the gas actually consumed).
- A portion stays in the **sender's wallet** (refund of unused gas).

---

## Chosen Implementation: Conditional Bypass

Zero-gas transactions are handled by **detecting** eligibility and then **skipping** cost checks, fee deduction, and refund. No credit, no undo, no stored amounts.

1. **Detect** — An early ante step (`AnteStepDetectZeroGas`) determines if the tx is zero-gas eligible: `tx.To` is in `ZeroGasActors.always_zero_gas_contracts` and `tx.Value == 0`. If so, it sets a **context marker** (empty `ZeroGasMeta{}` under `CtxKeyZeroGasMeta`). No state mutations; marker presence only.
2. **Skip in ante** — When the marker is set:
   - **Account verification** — From-address validation and account creation (if missing) still run in `AnteStepVerifyEthAcc`. Only the balance-vs-tx-cost check is skipped so first-time users with no gas balance can onboard.
   - `AnteStepMempoolGasPrice` — skip min gas price (CheckTx)
   - `AnteStepVerifyEthAcc` — skip only balance vs. tx cost (validation and account creation still run)
   - `AnteStepCanTransfer` — runs always (gas cap and value checks; value no-ops for zero-gas)
   - `AnteStepDeductGas` — skip fee deduction
3. **Skip in msg_server** — After execution, `RefundGas` is skipped when the marker is set (zero-gas txs were never charged, so nothing to refund).

Access: `evm.IsZeroGasEthTx(ctx)` returns true when the marker is set. No amounts (CreditedWei, PaidWei, RefundedWei) are stored.

### Account verification vs. gas bypass (functional requirement)

**Design requirement:** We only waive **gas payment** (balance vs. tx cost and deduction). All other ante behavior—especially **account verification**—must remain in place. This is an explicit functional requirement:

- **First-time onboarding:** A core use case is that a user with **no prior chain interaction** (no account, no gas balance) can submit their **first transaction ever** as a zero-gas call to an allowlisted contract. That is only possible if we still create the sender account when missing. If VerifyEthAcc were fully skipped, AnteStepIncrementNonce would run against a nil account and fail with "account is nil", blocking onboarding.
- **Consistency:** From-address validation and the existence of the sender account are required for correct nonce handling and for the rest of the ante chain. Skipping only gas-related checks (balance vs. cost, fee deduction) keeps security and consistency while enabling gasless execution.

Concretely:

- **VerifyEthAcc must not be fully skipped.** (1) From address must be valid. (2) Sender account must exist so AnteStepIncrementNonce can run. We still run validation and account creation; only the balance-vs-tx-cost check is skipped when zero-gas.
- **CanTransfer runs for zero-gas.** It enforces structural validity: (1) Gas fee cap vs. base fee passes because `EffectiveGasFeeCapWei` returns `max(baseFee, txGasCap)`. (2) Value transfer check no-ops when `tx.Value == 0` (eligibility requirement). Running the step ensures zero-gas txs are valid-looking transactions.

---

## How It Fits the Current Design

### Gas Payment Phases for Zero-Gas

For **zero-gas** txs, ante skips deduction and msg_server skips RefundGas. No charge, no refund. For normal txs, the usual flow applies: ante deducts, execution runs, msg_server calls RefundGas.

### Ante to Msg_server Communication

The ante step sets a **marker** (empty `ZeroGasMeta` in context under `CtxKeyZeroGasMeta`). The msg_server uses `evm.IsZeroGasEthTx(ctx)` to decide whether to run RefundGas. No amounts are passed; only presence of the marker matters.

---

## Implementation Plan

Implementation is tracked in **PLAN - Implementation Tracker** below. In brief: add context marker (const.go / evm.go); add `AnteStepDetectZeroGas` before MempoolGasPrice; in VerifyEthAcc skip only balance-vs-cost when zero-gas (validation and account creation still run), in DeductGas/MempoolGasPrice skip entire step when zero-gas; CanTransfer runs always; in msg_server skip RefundGas when zero-gas.

---

### Edge Cases

1. **Reverted transactions** — For zero-gas txs, RefundGas is not run, so there is nothing to undo. Execution can revert; the sender was never charged.
2. **Consensus errors** — If execution fails before the refund block, the framework reverts the tx. No special logic needed for zero-gas.
3. **Simulation** (`eth_estimateGas`, `eth_call`) — These skip ante and go straight to `ApplyEvmMsg`. No marker is set; behavior unchanged.

---

## Alternative Approach: Credit to Undo (Not Implemented)

An alternative design would temporarily credit the sender, run the normal pipeline (deduct, execute, refund), then undo the net gas payment (reclaim from fee collector and sender). That approach is **not** implemented. The code uses the conditional bypass above: detect zero-gas, then skip cost checks, deduction, and refund.

---

## Design Clarifications

**Q: What makes a transaction eligible for zero-gas execution?**  A: Eligibility
is determined purely by the EVM call target. If `tx.To` is in the
governance-controlled list **ZeroGasActors.always_zero_gas_contracts** (new `[]string`
field on x/sudo ZeroGasActors), the transaction is eligible—for **any sender**.
This makes the mechanism general-purpose: allowlist the Sai interface contract
for Sai-only behavior, or other contracts (e.g. bridging) as needed.

**Q: Who pays for gas in a zero-gas transaction?**  A: No one. Zero-gas txs are
never charged: ante skips the balance-vs-tx-cost check and fee deduction; the
msg_server skips RefundGas. CanTransfer still runs (gas cap and value checks).
Account verification (including account creation when missing) still runs. The
sender's balance is unchanged by gas.

**Q: Can the sender use funds for anything other than gas in a zero-gas tx?**  A: By
design, no. Sponsored transactions are expected to have `tx.Value == 0`, and
allowlisted contracts are trusted not to transfer native balance. While it is
theoretically possible for a mis-whitelisted contract to move funds, this is
treated as a governance or operational error rather than a design flaw.

**Q: How does ante-to-execution communication work?**  A: The ante step sets a
**marker** (empty `ZeroGasMeta` under `CtxKeyZeroGasMeta`) in the SDK context.
The context is preserved into the EthereumTx message handler during DeliverTx.
The msg_server uses `evm.IsZeroGasEthTx(ctx)` to decide whether to run RefundGas.
No amounts are stored or passed; only the presence of the marker matters.

**Q: What is the trust model of this system?**  A: The chain does not attempt to
sandbox or restrict contract behavior at runtime. Safety relies on
governance-controlled allowlisting of trusted contracts. Zero-gas execution is a
policy surface, not a permissionless mechanism.

--- 

## PLAN - Implementation Tracker

The gasless EVM flow has two phases: **ante** (detect zero-gas, set marker; later steps skip only gas-related checks and deduction) and **EthereumTx** (skip RefundGas when marker set). The `EthereumTx` msg handler runs only on DeliverTx and Simulate; on CheckTx, runMsgs does not execute messages. The marker is set in ante (CheckTx and DeliverTx) and is visible in msg_server for RefundGas skip.

### Design invariants and execution model

**Invariants (if these change, revisit the design):**

- [x] Eligibility: `tx.To ∈ ZeroGasActors.always_zero_gas_contracts` (EVM hex `[]string` on x/sudo), **any sender**; `tx.Value == 0`. No sender allowlist for EVM.
- [x] Zero-gas txs skip only **gas-related** checks and deduction: balance-vs-cost in VerifyEthAcc, DeductGas, RefundGas. CanTransfer runs. Account verification (from validation, account creation) still runs—see [Account verification vs. gas bypass](#account-verification-vs-gas-bypass-functional-requirement).
- [x] Context marker: empty `ZeroGasMeta` under `CtxKeyZeroGasMeta`; no stored amounts. Access via `evm.IsZeroGasEthTx(ctx)` / `evm.GetZeroGasMeta(ctx)`. Prereqs: `CtxKeyZeroGasMeta`, `GetZeroGasMeta`, `ZeroGasMeta` in const.go/evm.go; `always_zero_gas_contracts` in ZeroGasActors (proto, default, validation). Config: EVM ante uses `evmstate.Keeper.SudoKeeper.GetZeroGasEvmContracts`; SudoKeeper not in AnteHandlerOptions.

**Policy:** Sponsored txs pay no validator gas fees; no extra gas caps. Emergency disable: change ZeroGasActors (e.g. via governance) to remove or adjust `always_zero_gas_contracts`.

**Edge cases** (see [Edge Cases](#edge-cases), [Design Clarifications](#design-clarifications)): Reverted execution (no RefundGas run); execution not attempted (framework reverts); simulation (`eth_call`/`eth_estimateGas`) skips ante, no marker.

### Ante chain: detection and skip behavior

Detection must run **before** any step that would reject a zero-gas tx. Data: `tx.To` from `evm.UnpackTxData(msgEthTx.Data).GetTo()`; SDB shared. **Placement:** `AnteStepDetectZeroGas` after `AnteStepValidateBasic`, before `AnteStepMempoolGasPrice`.

- [x] **Detection:** `AnteStepDetectZeroGas` in [`evmante_zero_gas.go`](x/evm/evmante/evmante_zero_gas.go). Eligibility: `tx.To` in `GetZeroGasEvmContracts(ctx)`, `tx.Value == 0`. Set marker only: `sdb.SetCtx(sdb.Ctx().WithValue(evm.CtxKeyZeroGasMeta, &evm.ZeroGasMeta{}))`. No state mutations. Registered in [`all_evmante.go`](x/evm/evmante/all_evmante.go) after ValidateBasic, before MempoolGasPrice.
- [x] **Skip behavior when `IsZeroGasEthTx(sdb.Ctx())`:** MempoolGasPrice → skip. VerifyEthAcc → run validation and account creation; skip only balance-vs-tx-cost. CanTransfer → runs always. DeductGas → skip.

```
AnteStepSetupCtx → EthSigVerification → AnteStepValidateBasic
────────────────────────────────────────────────────────────
AnteStepDetectZeroGas   ← tx.To ∈ always_zero_gas_contracts, tx.Value == 0; set marker only
────────────────────────────────────────────────────────────
AnteStepMempoolGasPrice ← skips if zero-gas
AnteStepBlockGasMeter
AnteStepVerifyEthAcc    ← validation + account creation run; skip only balance vs. cost if zero-gas
AnteStepCanTransfer     ← runs always (gas cap, value; value no-ops when 0)
...
AnteStepDeductGas       ← skips if zero-gas
...
```

### Msg handler and testing

- [x] **Msg_server:** In [`msg_server.go`](x/evm/evmstate/msg_server.go), run RefundGas only when `!evm.IsZeroGasEthTx(rootCtxGasless)`. Validate `always_zero_gas_contracts` as EVM addresses in ZeroGasActors.Validate(); document: `nibid tx sudo edit-zero-gas '{"senders":[...],"contracts":[...],"always_zero_gas_contracts":["0x..."]}'`.
- [x] **Testing:** See [Test-to-Claim Mapping](#test-to-claim-mapping) for which tests validate each design claim. Unit: DetectZeroGas, VerifyEthAcc, DeductGas; integration: full ante, msg_server RefundGas skip.

### Test-to-Claim Mapping

The following table maps design claims to the tests that validate them. Use this as the canonical reference for regression safety.

| Claim | Test(s) | File |
|-------|---------|------|
| Eligibility: ineligible To → no marker | `TestAnteStepDetectZeroGas_NonEligible_NoMeta` | evmante_zero_gas_test.go |
| Eligibility: eligible To + value 0 → marker set | `TestAnteStepDetectZeroGas_Eligible_SetsMetaNoCredit` | evmante_zero_gas_test.go |
| Eligibility: eligible To + value ≠ 0 → no marker | `TestAnteStepDetectZeroGas_Eligible_NonZeroValue_NoMeta` | evmante_zero_gas_test.go |
| Marker set in CheckTx | `TestAnteStepDetectZeroGas_SetsMetaInContextWhenCheckTx` | evmante_zero_gas_test.go |
| No balance mutation (bypass approach) | All `TestAnteStepDetectZeroGas_*` tests | evmante_zero_gas_test.go |
| VerifyEthAcc: skip balance check for zero-gas; create account when missing | `TestVerifyEthAcc_ZeroGas_CreatesAccountWhenMissing`, `TestVerifyEthAcc` case "zero-gas: sender has no balance..." | evmante_zero_gas_test.go, evmante_can_transfer_test.go |
| DeductGas: skip when zero-gas | `TestAnteStepDeductGas_SkipsForZeroGasTx` | evmante_zero_gas_test.go |
| Normal tx: no zero-gas marker | `TestAnteHandlerEVM` "happy: signed tx...", `TestEthereumTx_ABCI` | all_evmante_test.go, msg_ethereum_tx_test.go |
| Zero-gas: full ante passes with no balance (first-time onboarding) | `TestAnteHandlerEVM` "zero-gas: sender with no balance..." | all_evmante_test.go |
| Zero-gas: meta populated after ante, min gas price bypassed | `TestAnteHandlerEVM` "zero-gas: allowlisted contract..." (MinGasPrices=1, gas price 0 or 1) | all_evmante_test.go |
| msg_server: RefundGas skipped; fee collector and sender unchanged | `TestMsgEthereumTx_ZeroGas`, `TestMsgEthereumTx_ZeroGas_WithRefund` | msg_ethereum_tx_test.go |
| Reverted execution: no deduction, no refund | `TestMsgEthereumTx_ZeroGas_Reverted` | msg_ethereum_tx_test.go |

**Implicit validations (no dedicated unit test):**

- **MempoolGasPrice skip:** `TestAnteHandlerEVM` "zero-gas: sender with no balance..." uses `MinGasPrices(1)` and gas price 0. MempoolGasPrice would reject if it ran; the test passes.
- **Simulation (eth_call/eth_estimateGas):** Design states simulation skips ante. Code path in `grpc_query.go` calls `ApplyEvmMsg` directly; no test explicitly asserts "eth_call with zero-gas-like tx behaves normally" but the architecture ensures ante is never invoked for these paths.

### Test Coverage Gaps

MempoolGasPrice zero-gas skip is validated implicitly via full ante chain tests in `all_evmante_test.go`. A dedicated unit test for this step could strengthen regression safety.

### Files and flow

| File | Role |
|------|------|
| `x/evm/const.go`, `x/evm/evm.go` | `CtxKeyZeroGasMeta`, `GetZeroGasMeta`, `ZeroGasMeta`, `IsZeroGasEthTx` |
| `proto/nibiru/sudo/v1/state.proto`, `x/sudo/` | `always_zero_gas_contracts`, DefaultZeroGasActors, Validate, getter |
| `x/evm/evmante/evmante_zero_gas.go` | `AnteStepDetectZeroGas` |
| `x/evm/evmante/all_evmante.go` | Insert DetectZeroGas in steps |
| `x/evm/evmante/evmante_can_transfer.go` | VerifyEthAcc: skip only balance check when zero-gas; CanTransfer: runs for zero-gas |
| `x/evm/evmante/evmante_gas_consume.go` | DeductGas: skip when zero-gas |
| `x/evm/evmante/evmante_mempool_fees.go` | MempoolGasPrice: skip when zero-gas |
| `x/evm/evmstate/msg_server.go` | Skip RefundGas when `IsZeroGasEthTx` |

**Flow:**

```
BaseApp.runTx
    ├── 1. Ante (CheckTx + DeliverTx): anteHandler sets marker; skip behavior as above; ante state persisted
    └── 2. runMsgs (DeliverTx + Simulate only): handler receives ctx with marker; skip RefundGas when set
```

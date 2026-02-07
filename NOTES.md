# Notes: EVM Ante Handler for Zero-Balance Gas Transactions

- [Question Posed](#question-posed)
- [Answer Summary](#answer-summary)
- [Relevant Code Locations](#relevant-code-locations)
  - [Entry Point: Ante Handler Routing](#entry-point-ante-handler-routing)
  - [EVM Ante Step Order](#evm-ante-step-order)
- [Checks That Need a Custom Bypass](#checks-that-need-a-custom-bypass)
  - [1. AnteStepVerifyEthAcc ΓÇö Balance vs. Tx Cost](#1-antestepverifyethacc--balance-vs-tx-cost)
  - [2. AnteStepCanTransfer ΓÇö Gas Cap and Value Transfer](#2-antestepcantransfer--gas-cap-and-value-transfer)
  - [3. AnteStepDeductGas ΓÇö Fee Deduction](#3-antestepdeductgas--fee-deduction)
  - [4. AnteStepMempoolGasPrice (Optional)](#4-antestepmempoolgasprice-optional)
- [Reference: Non-EVM Zero-Gas Pattern](#reference-non-evm-zero-gas-pattern)
- [Conceptual Model: Gas Payment Flow](#conceptual-model-gas-payment-flow)
- [Chosen Implementation: Credit ΓåÆ Normal Flow ΓåÆ Undo](#chosen-implementation-credit-%E2%86%92-normal-flow-%E2%86%92-undo)
  - [Flow](#flow)
  - [Why This Approach?](#why-this-approach)
- [How It Fits the Current Design](#how-it-fits-the-current-design)
  - [Gas Payment Phases](#gas-payment-phases)
  - [State Flow and SDB Lifecycle](#state-flow-and-sdb-lifecycle)
  - [Credit Source Options](#credit-source-options)
  - [Ante ΓåÆ Msg_server Communication](#ante-%E2%86%92-msg_server-communication)
- [Implementation Plan](#implementation-plan)
  - [Reclaim Logic (Undo Details)](#reclaim-logic-undo-details)
  - [Edge Cases](#edge-cases)
- [Alternative Approach: Conditional Bypass (Not Chosen)](#alternative-approach-conditional-bypass-not-chosen)
- [Implementation Tracker](#implementation-tracker)
  - [A. Core Design Invariants (Read Once, Then Treat as Law)](#a-core-design-invariants-read-once-then-treat-as-law)
  - [Ante Chain Ordering - When to Credit and Check To Address](#ante-chain-ordering----when-to-credit-and-check-to-address)
  - [B. Execution Model & Ordering Constraints](#b-execution-model--ordering-constraints)
  - [C. Economic & Accounting Decisions (Must Decide Before Coding)](#c-economic--accounting-decisions-must-decide-before-coding)
  - [D. Edge Case Matrix (Reason Through Before Coding)](#d-edge-case-matrix-reason-through-before-coding)
  - [Phase 1: Prerequisites](#phase-1-prerequisites)
  - [Phase 2: Ante Handler ΓÇö Credit Step](#phase-2-ante-handler--credit-step)
  - [Phase 3: EthereumTx Handler ΓÇö Undo](#phase-3-ethereumtx-handler--undo)
  - [F. Testing (Minimal but Sufficient)](#f-testing-minimal-but-sufficient)
  - [G. Files to Touch (Single Source of Truth)](#g-files-to-touch-single-source-of-truth)
- [Design Clarifications](#design-clarifications)

Date: 2026-02-05

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
// app/ante.go, lines 31-38
if !evm.IsEthTx(tx) {
    anteHandler = NewAnteHandlerNonEVM(keepers.PublicKeepers, options)
    return anteHandler(ctx, tx, sim)
}
anteHandler = evmante.NewAnteHandlerEvm(options)
return anteHandler(ctx, tx, sim)
```

- **File:** `app/ante.go`
- **Lines:** 31–38

### EVM Ante Step Order

The EVM ante handler is defined in `x/evm/evmante/all_evmante.go`:

```go
// x/evm/evmante/all_evmante.go (see current order in repo)
steps := []AnteStep{
    AnteStepSetupCtx,                    // outermost
    EthSigVerification,
    AnteStepValidateBasic,
    AnteStepMempoolGasPrice,             // ← mempool min gas price (CheckTx)
    AnteStepBlockGasMeter,
    AnteStepCreditZeroGas,               // ← zero-gas credit before balance check
    AnteStepVerifyEthAcc,                // ← balance >= tx cost
    AnteStepCanTransfer,                 // ← gas cap, value transfer
    AnteStepGasWanted,
    AnteStepDeductGas,                   // ← deduct fees from sender
    AnteStepIncrementNonce,
    AnteStepEmitPendingEvent,
    AnteStepFiniteGasLimitForABCIDeliverTx,
}
```

- **File:** `x/evm/evmante/all_evmante.go`

---

## Checks That Need a Custom Bypass

### 1. AnteStepVerifyEthAcc — Balance vs. Tx Cost

Validates that the sender’s balance covers the full transaction cost (fees + value).

| Location  | File                                 | Lines  |
|-----------|--------------------------------------|--------|
| Ante step | `x/evm/evmante/evmante_can_transfer.go` | 27–70  |
| Core check| `x/evm/evmstate/gas_fees.go`          | 95–116 |

**Key logic in `evmstate.CheckSenderBalance`:**

```go
cost := txData.Cost()
if balanceWei.ToBig().Cmp(cost) < 0 {
    return sdkioerrors.Wrapf(sdkerrors.ErrInsufficientFunds, ...)
}
```

Bypass: Skip this check or treat cost as zero when the tx is identified as the special type.

---

### 2. AnteStepCanTransfer — Gas Cap and Value Transfer

Validates:

1. `gasFeeCap >= baseFee`
2. If `value > 0`, balance >= value

| Location  | File                                 | Lines   |
|-----------|--------------------------------------|---------|
| Ante step | `x/evm/evmante/evmante_can_transfer.go` | 76–128  |

**Key checks:**

- Gas fee cap vs base fee: lines 104–110
- Value transfer vs balance: lines 115–126

Bypass: Skip or relax these checks for the special transaction type.

---

### 3. AnteStepDeductGas — Fee Deduction

Deducts transaction fees from the sender and sends them to the fee collector. Skips deduction when `fees.IsZero()`.

| Location     | File                                 | Lines    |
|--------------|--------------------------------------|----------|
| Ante step    | `x/evm/evmante/evmante_gas_consume.go` | 99–151   |
| Fee calc     | `x/evm/evmstate/gas_fees.go`          | 139–211 (VerifyFee) |
| Deduction    | `x/evm/evmstate/gas_fees.go`          | 119–137 (DeductTxCostsFromUserBalance) |

**Short-circuit when fees are zero:**

```go
// evmante_gas_consume.go, lines 123-126
if fees.IsZero() {
    return nil
}
```

Bypass: Either have `VerifyFee` return 0 for the special tx type, or add a conditional skip before calling `DeductTxCostsFromUserBalance`.

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

- `app/ante/fixed_gas.go`, lines 64–108
- `app/ante/deduct_fee.go`, lines 52–56, 158–168 (`isZeroGasMeter`)

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

## Chosen Implementation: Credit → Normal Flow → Undo

Instead of bypassing checks, we **simulate** zero gas cost by temporarily crediting the sender, letting the normal pipeline run, then undoing the net gas payment.

### Flow

1. **Credit** — When a free-gas tx is detected, credit the sender's balance with enough NIBI to pass all balance checks.
2. **Normal path** — Let ante, deduction, execution, and refund run unchanged. The credited balance passes checks; deduction and refund behave as usual.
3. **Measure** — After execution, we know how much gas was consumed and how much was refunded. The net cost to the sender is `actual_gas_cost = gasUsed × effectiveGasPrice`.
4. **Undo** — Transfer `actual_gas_cost` from the fee collector back to the sender (reversing the net gas payment), and reclaim any excess credited amount. The sender ends with zero net cost.

### Why This Approach?

1. **Existing logic unchanged** — All existing checks, deduction, and refund paths stay as-is. `CheckSenderBalance`, `DeductTxCostsFromUserBalance`, `RefundGas`, and EVM execution are untouched.
2. **Minimal code changes** — Add a credit step early in ante, add an undo step after refund in the msg handler.
3. **Consistent behavior** — Paths that bypass ante (e.g. `eth_call`, `eth_estimateGas`) are unaffected.
4. **Clear accounting** — Gas usage and cost are observable; only the net effect is reversed.
5. **Easy to disable** — Remove the credit and undo steps to revert to normal behavior.


---

## How It Fits the Current Design

### Gas Payment Phases

| Phase | Location | What Happens |
|-------|----------|--------------|
| **Ante: Deduct** | `evmante_gas_consume.go` → `DeductTxCostsFromUserBalance` | Full cost (`gasLimit × effectiveGasPrice`) deducted from sender, added to fee collector |
| **Execution** | `msg_server.ApplyEvmMsg` | EVM runs; `gasUsed` is determined |
| **Refund** | `msg_server.go` (after execution, before post_execution_events) → `RefundGas`; then msg_server sets `ZeroGasMeta.RefundedWei` | `leftoverGas = gasLimit - gasUsed` refunded: fee collector → sender |

Net effect for the sender: `-full_cost + refund = -(gasUsed × effectiveGasPrice) = -actual_gas_cost`.

The undo step reverses this: transfer `actual_gas_cost` from fee collector back to sender, so the sender's net change is zero.

### State Flow and SDB Lifecycle

- **Ante** creates an SDB, runs steps, and calls `sdb.Commit()` on DeliverTx. Credit and deduction happen in this SDB and are persisted.
- **Msg_server** creates a new SDB from the same root context. It reads the committed state, so it sees the credited balance and deduction.

Credit in ante and undo in msg_server both operate on the same chain state.

### Credit Source Options

**Chosen model:** Chain-minted temporary credit (ante adds balance; undo burns from fee collector and sender). See Plan 4.4 and Plan 5.

**Alternatives not chosen:** (1) Fee collector loan — debit fee collector, credit sender. (2) Subsidy module account — dedicated prefunded module. Both were superseded by chain-minted credit.

### Ante → Msg_server Communication

The msg_server must know that a tx was a zero-gas tx and how much was credited. The ante stores a `*ZeroGasMeta` under `CtxKeyZeroGasMeta` when crediting; msg_server reads it via `evm.GetZeroGasMeta(ctx)` before running the undo.

---

## Implementation Plan

Implementation is tracked in **PLAN - Implementation Tracker** below. In brief: add `CtxKeyZeroGasMeta` and `ZeroGasMeta` (const.go / evm.go); add `AnteStepCreditZeroGas` in evmante; after `RefundGas` in msg_server, read meta, set `RefundedWei`, then run undo (reclaim) when implemented. See Plan 7 (prereqs), Plan 8 (credit), Plan 9 (undo), and Plan 11 (files).

---

### Reclaim Logic (Undo Details)

After refund, the sender holds:
- Original balance
- Plus credit X
- Minus full_cost (deduction)
- Plus refund

So: `sender_balance = original + X - full_cost + refund = original + X - actual_gas_cost`.

To achieve zero net cost for the sender:
- Reclaim `(X - actual_gas_cost)` from sender back to the credit source (fee collector or subsidy pool).
- Sender ends with `original`, as if they never paid gas.

---

### Edge Cases

1. **Reverted transactions** — `RefundGas` still runs on revert. Undo should run as well so the sender is not charged.
2. **Consensus errors** — If execution fails before `RefundGas`, no refund occurs. Undo must still run and must not assume a refund was applied.
3. **Credit amount** — `X` must be at least `txData.Cost()` (which includes `value` if present). A small buffer may be needed for rounding.
4. **Simulation** (`eth_estimateGas`, `eth_call`) — These skip ante and go straight to `ApplyEvmMsg`. No credit is applied; no special handling for undo is needed.

---

## Alternative Approach: Conditional Bypass (Not Chosen)

A different approach would add conditional bypass logic to existing steps:

1. Add an early detection step that identifies the special tx type and sets a context flag.
2. In each relevant step, skip the check when the flag is set:
   - `AnteStepVerifyEthAcc` — skip `CheckSenderBalance`
   - `AnteStepCanTransfer` — skip gas cap and value checks
   - `AnteStepDeductGas` — skip deduction or have `VerifyFee` return 0
   - `AnteStepMempoolGasPrice` — skip for flagged txs

The credit-then-undo approach was chosen because it avoids modifying core logic and keeps all existing paths intact.

---

## Design Clarifications

**Q: What makes a transaction eligible for zero-gas execution?**  A: Eligibility
is determined purely by the EVM call target. If `tx.To` is in the
governance-controlled list **ZeroGasActors.always_zero_gas_contracts** (new `[]string`
field on x/sudo ZeroGasActors), the transaction is eligible—for **any sender**.
This makes the mechanism general-purpose: allowlist the Sai interface contract
for Sai-only behavior, or other contracts (e.g. bridging) as needed.

**Q: Who pays for gas in a zero-gas transaction?**  A: No party ultimately pays.
The system temporarily mints native tokens to allow the transaction to pass
normal balance checks and fee deduction. After execution, the exact gas cost is
known and the net effect is reversed: the minted funds are reclaimed from both
the sender and the fee collector. The sender ends with their original balance,
the fee collector ends unchanged, and total supply is preserved.

**Q: Can the credited balance be used for anything other than gas?**  A: By
design, no. Sponsored transactions are expected to have `tx.Value == 0`, and
allowlisted contracts are trusted not to transfer native balance. While it is
theoretically possible for a mis-whitelisted contract to move funds, this is
treated as a governance or operational error rather than a design flaw.

**Q: How is gas cost computed for undo?**  A: The undo step uses the same
effective gas price and gas usage already computed by the EVM execution path. The
reclaimed amount is `credited_amount - actual_gas_cost`, ensuring the sender’s
net balance change is zero.

**Q: How does ante-to-execution communication work?**  A: The ante handler stores
a `*ZeroGasMeta` (CreditedWei, PaidWei, RefundedWei) under `CtxKeyZeroGasMeta` in
the SDK context using `ctx.WithValue`. This context is preserved into the
EthereumTx message handler during DeliverTx, allowing the undo logic to execute
with full information.

**Q: What is the trust model of this system?**  A: The chain does not attempt to
sandbox or restrict contract behavior at runtime. Safety relies on
governance-controlled allowlisting of trusted contracts. Zero-gas execution is a
policy surface, not a permissionless mechanism.

--- 

## PLAN - Implementation Tracker

Use this checklist and tracking section to implement the gasless EVM transaction
flow. The flow runs in two phases: **ante** (credit) and **EthereumTx** (undo).

**Important:** `EthereumTx` msg handler runs only on DeliverTx and Simulate. On
CheckTx, runMsgs exits early without executing messages. So credit must run in
ante (both CheckTx and DeliverTx), but undo runs only in EthereumTx
(DeliverTx/Simulate).

### Plan 1 - Core Design Invariants (Read Once, Then Treat as Law)

These are the assumptions the implementation relies on. If any change, the design
must be revisited.

- [x] Zero-gas eligibility is determined **only** by `tx.To ∈ ZeroGasActors.always_zero_gas_contracts` (EVM only for now; **any sender**). The `always_zero_gas_contracts` field is `[]string` (EVM hex addresses) on the existing x/sudo ZeroGasActors struct.
- [x] Sponsored transactions are expected to have `tx.Value == 0`
- [ ] Open contracts are trusted not to transfer native balance
- [ ] Zero-gas execution is a **policy surface**, not a sandbox
- [ ] Temporary mint → settle → reclaim leaves **total supply unchanged**
- [ ] Sender balance after undo equals sender balance before tx
- [x] Credit is allowed to mutate state during CheckTx when necessary to pass later balance validation; economic neutrality is intended to be restored during DeliverTx inside the `EthereumTx` message handler in `x/evm/evmstate/msg_server.go` via an undo step.
- [ ] Undo must always run for DeliverTx where credit was applied (or safely no-op)

### Plan 2 - Ante Chain Ordering - When to Credit and Check To Address

The credit step and the to-address check (`tx.To ∈ always_zero_gas_contracts`) must run **early enough** that the tx is not rejected during CheckTx, but **late enough** that we have the data we need. For EVM zero gas we do **not** check sender—only that the call target is in `always_zero_gas_contracts`.

**When does each piece of data become available?**

- **From (sender)** — After `EthSigVerification`. `msgEthTx.From` is set by that step (see `evmante_sigverify.go` line 56: `msgEthTx.From = sender.Hex()`). Before that step, From may be empty.
- **To (contract)** — From the start. Use `txData.GetTo()` from `evm.UnpackTxData(msgEthTx.Data)`. The To address is in the tx payload; no ante step sets it.
- **SDB** — From the start. Created before the step loop. All steps share the same SDB; balance changes are visible to subsequent steps.

**Earliest we can run the credit step:** After `EthSigVerification` (we need From only for logging or future use; eligibility is `tx.To ∈ always_zero_gas_contracts` only).

**Latest we must run the credit step:** Before `AnteStepVerifyEthAcc` (that step calls `CheckSenderBalance` and would reject a zero-balance sender).

**Recommended placement:** Insert `AnteStepCreditZeroGas` between `AnteStepBlockGasMeter` and `AnteStepVerifyEthAcc`:

```
AnteStepSetupCtx
EthSigVerification          ← From is set here
AnteStepValidateBasic
AnteStepMempoolGasPrice
AnteStepBlockGasMeter
─────────────────────────
AnteStepCreditZeroGas       ← tx.To ∈ always_zero_gas_contracts check + add to sender balance (both here)
─────────────────────────
AnteStepVerifyEthAcc        ← Balance check (passes because of credit)
AnteStepCanTransfer
...
```

Both the **to-address check** (`tx.To ∈ always_zero_gas_contracts`; no sender check) and
the **credit** (add X to sender via `sdb.AddBalance`) happen in the same step.
This runs on both CheckTx and DeliverTx, so the balance check passes in both
phases.

### Plan 3 - ZeroGasMeta (context payload)

A single context value under **`CtxKeyZeroGasMeta`** carries the three amounts. Which amounts are set indicates what has already happened (credit only; credit + deduct; or full path including refund), so undo can branch safely when execution short-circuits.

**`ZeroGasMeta` struct** — Fields: `CreditedWei`, `PaidWei`, `RefundedWei` only (no Phase). Who sets what:

- **Credit step (ante):** Sets `CreditedWei`; `PaidWei`/`RefundedWei` remain unset (nil).
- **DeductGas (ante):** If meta exists, set `PaidWei` to the deducted amount.
- **After RefundGas (msg_server):** Set `RefundedWei`; then run undo using the three numbers.

**Inference:** Undo logic branches on which amounts are non-nil (e.g. `RefundedWei != nil` means refund ran; `PaidWei == nil` means deduction did not run).

**Access:** Helper **`evm.GetZeroGasMeta(ctx)`** in the evm package returns `*ZeroGasMeta` or `nil`; use it in ante steps and in EthereumTx handler instead of raw `ctx.Value(...)`.

### Plan 4 - Execution Model & Ordering Constraints

These constraints determine *where* code can safely live.

- [x] Confirm that these are true:
  - [x] Ante state commits are visible to `EthereumTx` handler
  - [x] `ctx.WithValue` persists from ante into msg execution
- [x] Credit step placement:
  - [x] Runs **after** `EthSigVerification`. `From` available only after `EthSigVerification`
  - [x] Runs **before** `AnteStepVerifyEthAcc`. `To` available from tx payload (no ante dependency)
  - [x] Eligibility check runs in CheckTx
  - [x] Balance mutation can occur in both CheckTx and DeliverTx, but only DeliverTx persists state via `sdb.Commit()`.

- [x] **4.1** Add context key and `GetZeroGasMeta(ctx)` in [`x/evm/const.go`](x/evm/const.go); define `ZeroGasMeta` struct (CreditedWei, PaidWei, RefundedWei) in `x/evm/evm.go`. Use `evm.GetZeroGasMeta(ctx)` to read; reference: existing `CtxKeyEvmSimulation` and `IsSimulation`.

- [x] **4.2** Detection criteria for "zero-gas tx" **(decided)**
  - Use **Option A:** extend `x/sudo` ZeroGasActors with a new field
  **`always_zero_gas_contracts []string`** (EVM hex addresses). Eligibility:
  `tx.To != nil && tx.To.Hex() ∈ always_zero_gas_contracts`; **any sender** gets
  zero gas when calling these contracts. Existing `Senders` and `Contracts`
  remain for Wasm (restricted); `always_zero_gas_contracts` is EVM-only and
  open-to-all. Requires proto/state change in sudo and default/migration for
  existing chains.

- [x] **4.3** Access sudo zero-gas configuration via the EVM state keeper
  - Final design decision: EVM ante code reads zero-gas configuration through `evmstate.Keeper.SudoKeeper` (and the `GetZeroGasEvmContracts` helper). We will **not** wire `SudoKeeper` into `AnteHandlerOptions`; the EVM state keeper remains the single integration point for sudo configuration in the EVM ante stack.

- [x] **4.4** Decide credit source
  - Chosen model: **chain-minted temporary credit (loan)** — the ante step increases the sender's balance without debiting an existing account; later undo logic will claw back the net gas cost from both the fee collector and the sender's post-refund balance so total supply and long-run balances remain unchanged.

### Plan 5 - Economic & Accounting Decisions (Must Decide Before Coding)

- [x] Choose credit source  
  Chosen model: **chain-minted temporary credit (loan)** — ante increases the sender's balance without debiting an existing account; later undo logic claws back the net gas cost from both the fee collector and the sender so total supply and long-run balances remain unchanged.

- [x] Document reclaim direction explicitly  
  Flow of funds: (1) temporarily mint/loan the needed amount to the sender; (2) sender uses that credit to pay gas, which flows into the fee collector; (3) EVM may refund some gas back to the sender; (4) undo logic then burns all of the temporarily minted credit by (a) burning the portion that ended up in the fee collector and (b) burning the portion that ended up back in the sender’s balance (refund plus any unspent credit), using `ZeroGasMeta` to account for each leg so total supply and both balances return to their pre-tx levels.

- [x] Document validator fee semantics for sponsored txs  
  Sponsored EVM txs pay **no validator gas fees**; validators do not earn per-tx gas from these calls. Sai usage (e.g. perps DEX fees in USDC flowing to protocol revenue / NIBI buyback and burn) is treated as the indirect economic compensation path for validators and token holders instead of direct gas payment on these txs.

- [x] Decide global / per-contract sponsored gas caps  
  No extra zero-gas–specific caps are introduced. Sponsored txs are bounded by the existing EVM / block gas limits and external operational processes; the design does not add separate global or per-contract gas caps for this feature.

- [x] Decide emergency disable mechanism  
  Emergency disable is handled operationally by **editing sudo ZeroGasActors state** (e.g. via multisig/governance) to remove or adjust `always_zero_gas_contracts`. No separate on-chain kill-switch parameter is added; the allowlist itself is the kill switch, and this behavior is documented as part of the design.

### Plan 6 - Edge Case Matrix (Reason Through Before Coding)

- [ ] Reverted EVM execution
- [ ] EVM execution not attempted
- [ ] `RefundGas` not executed
- [ ] `evmResp == nil`
- [ ] Simulation path (`eth_call`, `eth_estimateGas`)
- [ ] Ensure undo behavior is correct or a safe no-op in all cases

### Plan 7 - Prerequisites in x/evm and x/sudo

- [x] Add `CtxKeyZeroGasMeta` and `GetZeroGasMeta(ctx)` in `x/evm/const.go`; add `ZeroGasMeta` struct in `x/evm/evm.go`
- [x] Add `always_zero_gas_contracts` to ZeroGasActors (proto + default + validation)

### Plan 8 - Phase 2: Ante Handler — Credit Step

- [x] Create `AnteStepCreditZeroGas`
- [x] Implement eligibility detection: `tx.To ∈ GetZeroGasActors(ctx).AlwaysZeroGasContracts` (no sender check)
- [x] Enforce `tx.Value == 0`
- [x] Persist `*ZeroGasMeta` (CreditedWei set) in context under `CtxKeyZeroGasMeta`
- [x] Register step between:
  - `AnteStepBlockGasMeter`
  - `AnteStepVerifyEthAcc`

- [x] **8.1** Create `AnteStepCreditZeroGas` in new file [`x/evm/evmante/evmante_zero_gas.go`](x/evm/evmante/evmante_zero_gas.go)
  - Signature: `func(sdb *evmstate.SDB, k *evmstate.Keeper, msgEthTx *evm.MsgEthereumTx, simulate bool, opts AnteOptionsEVM) error`
  - Must run **before** `AnteStepVerifyEthAcc` (so credit is present when balance is checked)

- [x] **8.2** Implement detection in `AnteStepCreditZeroGas`
  - Get `ZeroGasActors` from SudoKeeper; if `len(AlwaysZeroGasContracts) == 0`, return `nil` (no-op)
  - Parse `msgEthTx`: `txData.GetTo()` (contract; nil for deploy). Contract deploys (`To == nil`) are not eligible.
  - Check: `To != nil && To.Hex() ∈ always_zero_gas_contracts` (compare against AlwaysZeroGasContracts list; normalize hex if needed). No sender check.
  - If not matched, return `nil`

- [x] **8.3** Compute credit amount X
  - Current design uses a **fixed credit of 10 NIBI (in unibi → wei)** instead of `txData.Cost()`, via `evm.NativeToWei(new(big.Int).Mul(big.NewInt(10), big.NewInt(1_000_000)))`.
  - Ensure X > 0; if the computed fixed amount is non-positive, skip credit.

- [x] **8.4** Apply credit in SDB
  - Apply credit to the sender using `sdb.AddBalance(fromAddr, creditWeiU256, tracing.BalanceChangeTransfer)`.
  - Precise long-term credit source (fee collector vs. subsidy module) remains a policy decision tracked under Plan 5; current implementation behaves as a temporary balance boost for the sender.

- [x] **8.5** Set context for msg handler
  - `meta := &evm.ZeroGasMeta{CreditedWei: X}`; `sdb.SetCtx(sdb.Ctx().WithValue(evm.CtxKeyZeroGasMeta, meta))`
  - msg_server reads via `evm.GetZeroGasMeta(ctx)` and uses `meta.CreditedWei`, `meta.PaidWei`, `meta.RefundedWei`

- [x] **8.6** Register step in ante chain
  - Edit [`x/evm/evmante/all_evmante.go`](x/evm/evmante/all_evmante.go)
  - Insert `AnteStepCreditZeroGas` **after** `AnteStepBlockGasMeter` and **before** `AnteStepVerifyEthAcc` (so it runs before balance checks)

### Plan 9 - Phase 3: EthereumTx Handler — Undo

- [x] Identify guaranteed post-execution insertion point
- [x] Read zero-gas metadata from context
- [x] Compute burn amounts via `meta.AmountsToUndoCredit()` (no manual `actualGasCost` in handler)
- [x] Reclaim by burning from fee collector and sender (SubBalance only)
- [ ] Ensure undo:
  - [ ] Runs on success
  - [ ] Runs on revert
  - [ ] Safely no-ops if execution did not occur

- [x] **9.1** Locate insertion point in [`x/evm/evmstate/msg_server.go`](x/evm/evmstate/msg_server.go)
  - After `RefundGas`, before `stage = "post_execution_events_and_tx_index"` (see msg_server.go; RefundGas and RefundedWei population are in that block).
  - At this point: `evmResp.GasUsed`, `evmMsg.From`, `evmMsg.GasLimit` are available

- [x] **9.2** Read zero-gas metadata from context
  - `meta := evm.GetZeroGasMeta(rootCtxGasless)` in the same block where `RefundedWei` is set.
  - If `meta == nil`, skip undo (not a zero-gas tx)

- [x] **9.3** Set `meta.RefundedWei` to refund wei (`refundGas * weiPerGas`) after RefundGas (implemented).
- [x] **9.3** Obtain burn amounts via `meta.AmountsToUndoCredit()` — returns `feeCollectorBurnWei`, `txSenderBurnWei`; no manual `actual_gas_cost` computation in the handler.

- [x] **9.4** Implement reclaim logic (implemented as burns)
  - Reclaim is implemented as **burns** (SubBalance only, no AddBalance): burn from fee collector, burn from sender.
  - **Defensive rule:** Before each `SubBalance`, get `bal := sdb.GetBalance(addr)`. If `bal.Cmp(amount) < 0`, subtract `bal` (zero out); otherwise subtract `amount`.
  - Use **`tracing.BalanceChangeTransfer`** for both SubBalance calls.
  - Call `sdb.Commit()` after the balance updates.
  - Use same SDB as RefundGas (`sdb` in EthereumTx). Formula: same as **Reclaim Logic (Undo Details)** above.

- [ ] **9.5** Handle edge cases
  - **Reverted tx:** RefundGas still runs; run undo so sender is not charged
  - **Consensus error before RefundGas:** `evmResp` may be nil; guard undo with `evmResp != nil`
  - **Simulate:** Undo runs; ensure it doesn’t break simulation semantics

- [x] **10.1** Validate `always_zero_gas_contracts` entries as EVM addresses in ZeroGasActors.Validate() (same pattern as existing Contracts in [`x/sudo/msgs.go`](x/sudo/msgs.go), e.g. QueryEthAccountRequest or equivalent).

- [x] **10.2** Document how to add EVM open contracts
  - Same msg as today: `nibid tx sudo edit-zero-gas '{"senders":[...],"contracts":[...],"always_zero_gas_contracts":["0x..."]}'`
  - `always_zero_gas_contracts`: EVM hex addresses (0x...). Any sender gets zero gas when calling these.

### Plan 10 - Testing (Minimal but Sufficient)

- [ ] Unit: Ante credit step
- [ ] Unit: Undo math
- [ ] Integration: zero-balance sender, allowlisted contract
- [ ] Regression: normal EVM tx unchanged
- [ ] Regression: `eth_call` / `eth_estimateGas` unchanged

- [ ] **10.3** Unit test: `AnteStepCreditZeroGas`
  - Tx.To not in always_zero_gas_contracts → no credit, no context value
  - Tx.To in always_zero_gas_contracts → credit applied, context value set
  - Insufficient credit source balance → error

- [ ] **10.4** Unit test: Undo in EthereumTx
  - Mock context with `CtxKeyZeroGasMeta` and `*ZeroGasMeta`; verify reclaim math
  - Reverted tx: verify undo runs and sender balance correct

- [ ] **10.5** Integration test: full flow
  - Fund subsidy source (or fee collector)
  - Add contract to ZeroGasActors.always_zero_gas_contracts
  - Send EVM tx from any sender to that contract with zero balance for gas
  - Assert: tx succeeds, sender balance unchanged (modulo value/calls)

- [ ] **10.6** Regression: non-zero-gas txs
  - Ensure normal EVM txs unchanged
  - Ensure `eth_call` / `eth_estimateGas` (no ante) unchanged


### Plan 11 - Files to Touch (Single Source of Truth)

| File | Role |
|------|------|
| `x/evm/const.go` | Context key and `GetZeroGasMeta(ctx)`; `ZeroGasMeta` struct in `x/evm/evm.go` |
| `proto/nibiru/sudo/v1/state.proto` | Add `always_zero_gas_contracts` to ZeroGasActors; regenerate |
| `x/sudo/` | DefaultZeroGasActors, Validate (always_zero_gas_contracts), getter; migration if needed |
| `x/evm/evmante/evmante_zero_gas.go` | Credit step and ante registration |
| `x/evm/evmante/all_evmante.go` | Insert step in `steps` slice |
| `x/evm/evmstate/msg_server.go` | RefundedWei population + undo (reclaim) implemented |

- EVM ante reads zero-gas config via `evmstate.Keeper.SudoKeeper` only; `SudoKeeper` is not added to `AnteHandlerOptions` (see Plan 4.3).

Flow Overview

```
BaseApp.runTx
    │
    ├── 1. Ante (CheckTx + DeliverTx)
    │       anteCtx = cacheTxContext(ctx)
    │       newCtx = anteHandler(anteCtx, tx)
    │       ctx = newCtx.WithMultiStore(ms)
    │       msCache.Write()  ← ante state persisted
    │
    └── 2. runMsgs (DeliverTx + Simulate only; NOT CheckTx)
            runMsgCtx = cacheTxContext(ctx)  ← inherits ante's context
            handler(runMsgCtx, msg)          ← EthereumTx receives ctx with CtxKeyZeroGasMeta
            msCache.Write()  ← msg handler state persisted
```

- [ ] Setup and helpers
  - [ ] Initialize TestDeps to get BankKeeper and a fresh sdk.Context
  - [ ] Create two accounts (alice, bob); use EVM helpers where convenient
  - [ ] Helper asserts: GetWeiBalance, bank balance for `unibi`, infer wei-store as needed

- [ ] GetWeiBalance basics
  - [ ] New accounts return 0
  - [ ] After crediting only `unibi`, returns `unibi * 10^12` wei
  - [ ] After setting only wei-store (< 1e12), returns wei-store value

- [ ] AddWei behavior
  - [ ] No-op for nil and zero amounts
  - [ ] Below threshold (< WeiPerUnibi): updates only wei-store; `unibi == 0`
  - [ ] Exactly threshold (== WeiPerUnibi): mints 1 `unibi`, wei-store -> 0
  - [ ] Crossing threshold with multiple adds: `unibi += k`, wei-store `= r`
  - [ ] WeiBlockDelta increases by added wei per call in same block
  - [ ] Emits bank.EventWeiChange with reason AddWei

- [ ] SubWei behavior
  - [ ] No-op for nil and zero amounts
  - [ ] From wei-store only (wei-store >= amt): deduct wei-store; `unibi` unchanged
  - [ ] Pull from `unibi` when wei-store < amt and aggregate >= amt; correct remainder
  - [ ] Edge: subtract exactly aggregate -> both `unibi` and wei-store zero
  - [ ] Edge: subtract to leave only wei-store residue
  - [ ] Insufficient funds: error; no state or delta mutation
  - [ ] WeiBlockDelta decreases by subtracted wei
  - [ ] Emits bank.EventWeiChange with reason SubWei

- [ ] Cross-method invariants
  - [ ] Add then Sub same amount (below/over threshold) restores original state
  - [ ] Add A + Add B then Sub (A+B) restores original state
  - [ ] Multiple accounts remain independent (alice unaffected by bob)

- [ ] setNibiBalanceFromWei pathway
  - [ ] For wei < WeiPerUnibi: sets `unibi == 0`, wei-store = wei
  - [ ] For wei == k*WeiPerUnibi + r: `unibi == k`, wei-store `== r`

- [ ] Parse helpers
  - [ ] ParseNibiBalance(wei): correct `(wei/1e12, wei%1e12)` for 0, 1, 1e12-1, 1e12, 1e12+1, multiples+remainder
  - [ ] ParseNibiBalanceFromParts(unibi, wei): normalizes to same `(unibi, wei)` after round-trip
  - [ ] BigToU256Safe: accepts 0, small, 2^256-1; rejects negative and >= 2^256

- [ ] Supply tracking (if accessible)
  - [ ] weiStoreSupply increases on AddWei, decreases on SubWei
  - [ ] weiStoreSupply equals sum of wei-store balances across accounts

- [ ] WeiBlockDelta consistency
  - [ ] Sequence Add x, Add y, Sub z -> delta == x + y - z
  - [ ] (Optional) Next-block reset behavior if harness supports transient reset

- [ ] Error and deletion edge-cases
  - [ ] setWeiStoreBalance deletes key when new balance is zero (no residual)
  - [ ] No underflow on SubWei happy paths

- [ ] Event coverage
  - [ ] bank.EventWeiChange fires for Add/Sub
  - [ ] (Optional) eventsForSendCoins emits wei-change when `unibi` present on SendCoins

- [ ] Test names (indicative)
  - [ ] TestWei_GetWeiBalance_Basics
  - [ ] TestWei_AddWei_BelowThreshold
  - [ ] TestWei_AddWei_ExactlyThreshold_MintsUnibi
  - [ ] TestWei_AddWei_CrossThreshold_MultiAdds
  - [ ] TestWei_SubWei_FromWeiStoreOnly
  - [ ] TestWei_SubWei_PullsFromUnibi
  - [ ] TestWei_SubWei_InsufficientFunds
  - [ ] TestWei_WeiBlockDelta_Accumulation
  - [ ] TestWei_ParseNibiBalance
  - [ ] TestWei_BigToU256Safe
  - [ ] (Optional) TestWei_SendCoins_Unibi_EmitsWeiChange

- [ ] Out-of-scope
  - [ ] Gas parity/perf; complex multi-block sweeping unless trivial



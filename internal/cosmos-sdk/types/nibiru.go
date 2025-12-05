package types

// ---------------------------------------------------------
// New EVM
// ---------------------------------------------------------

// lastErrApplyEvmMsg: Context keys should be unexported, unique types. A named,
// empty struct is guaranteed to be unique in any Go scope.
type lastErrApplyEvmMsg struct{}

// Holds a reference to the latest non-nil error resulting from apply
// EVM, persisting through panics and nested "revert" calls. This
// facilitates the propagation of low-level VM errors.
func (c Context) LastErrApplyEvmMsg() error {
	// Get/Set helpers. Value receiver is fine: it mutates the pointed slot.
	if c.lastErrApplyEvmMsg == nil {
		return nil
	}
	return c.lastErrApplyEvmMsg.evmErr
}
func (c Context) WithLastErrApplyEvmMsg(e error) Context {
	if c.lastErrApplyEvmMsg == nil {
		return c
	}
	c.lastErrApplyEvmMsg.evmErr = e
	return c
}

// True if the current execution context is an EVM transaction
func (c Context) IsEvmTx() bool { return c.isEvmTx }

// WithIsEvmTx specifies whether the current execution context is an EVM transaction
func (c Context) WithIsEvmTx(isEvmTx bool) Context {
	c.isEvmTx = isEvmTx
	return c
}

// EvmTxHash returns the tx hash of the current Ethereum transaction.
func (c Context) EvmTxHash() [32]byte { return c.evmTxHash }

// WithEvmTxHash sets the tx hash of the current Ethereum transaction.
func (c Context) WithEvmTxHash(txhash [32]byte) Context {
	c.evmTxHash = txhash
	return c
}

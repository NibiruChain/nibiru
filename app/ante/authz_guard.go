package ante

// TODO: https://github.com/NibiruChain/nibiru/issues/1915
// feat(ante): Add an authz guard to disable authz Ethereum txs and provide
// additional security around the default functionality exposed by the module.
//
// Implemenetation Notes
// UD-NOTE - IsAuthzMessage fn. Use authz import with module name
// UD-NOTE - Define set of disabled txMsgs

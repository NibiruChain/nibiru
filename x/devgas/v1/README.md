# devgas

Module `x/devgas` is retired. It paid a contract's deployer a share of the gas
fees on transactions that called the contract. Zero-gas transactions pay no
fee, so there is nothing to share.

- The transaction messages `MsgRegisterFeeShare`, `MsgUpdateFeeShare`,
  `MsgCancelFeeShare`, and `MsgUpdateParams` fail with error
  `ErrDevgasDeprecated`.
- The ante handler pays no fee share. Fees stay in the fee collector.
- The gRPC queries and `nibid query devgas` still read the module's store.
- Keeper method `ClearFeeShareState` deletes the registered fee shares and
  sets module param `enable_fee_share` to false. A chain upgrade handler has
  to call it.

The protobuf types stay registered so that historical transactions decode.

---
order: 420
---

# Gonibi (Golang SDK)

Golang Client SDK for interacting with the Nibiru blockchain. Gonibi is useful
for building applications on top of the chain without needing to have an in-depth
knowledge on the underlying module structure.
{synopsis}

- [Full Go Reference Docs: gonibi](https://pkg.go.dev/github.com/Unique-Divine/gonibi)
- Repo: [NibiruChain/nibiru/gosdk](https://github.com/NibiruChain/nibiru/tree/main/gosdk)

## Installation

To use the package, add github.com/NibiruChain/nibiru as a dependency in your go.mod file. To get the latest gosdk with the most up-to-date features, include the latest commit in your dependency via the following command:

```bash
go get github.com/NibiruChain/nibiru@cc4ddd4b51317ff7a4faa251b97e72855456c8e6
```

To ensure compatibility of your project with external dependencies, include the following go.mod replace directive:

```go
replace (
  cosmossdk.io/api => cosmossdk.io/api v0.3.1

  github.com/cosmos/iavl => github.com/cosmos/iavl v0.20.0

  github.com/ethereum/go-ethereum => github.com/NibiruChain/go-ethereum v1.10.27-nibiru
  github.com/gogo/protobuf => github.com/regen-network/protobuf v1.3.3-alpha.regen.1

  github.com/linxGnu/grocksdb => github.com/linxGnu/grocksdb v1.8.12

  // pin version! 126854af5e6d has issues with the store so that queries fail
  github.com/syndtr/goleveldb => github.com/syndtr/goleveldb v1.0.1-0.20210819022825-2ae1ddf74ef7

  // stick with compatible version or x/exp in v0.47.x line
  golang.org/x/exp => golang.org/x/exp v0.0.0-20230711153332-06a737ee72cb
)
```

## Connecting to gRPC

To connect to the Nibiru gRPC endpoint, use the following template: `grpc.nibiru/fi:443`. For the testnet, use: `grpc.testnet-1.nibiru/fi:443`.

```golang
  grpcConn, err := gosdk.GetGRPCConnection("grpc.testnet-1.nibiru.fi:443", false, 10)
  if err != nil {
    log.Fatalf("Failed to create grpc connect: %v", err)
  }
```

## Setting up imports

```golang
import (
  "context"
  "encoding/json"
  "fmt"
  "log"
  "time"

  wasm "github.com/CosmWasm/wasmd/x/wasm/types"
  gosdk "github.com/NibiruChain/nibiru/gosdk"
  tokenfactory "github.com/NibiruChain/nibiru/x/tokenfactory/types"
  sdk "github.com/cosmos/cosmos-sdk/types" // Import cosmos SDK types
)
```

## Querier

The Querier allows you to query data from the Nibiru blockchain. Here is an example of how to use the Querier:

```go
  // Querier
  querier, err := gosdk.NewQuerier(grpcConn)
  if err != nil {
    log.Fatalf("Failed to create nibiru querier: %v", err)
  }
  ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
  defer cancel()
```

### Querying a Smart Contract State

You can query the state of a smart contract using the following example:

```go
  contractAddress := "" // some contract address

  // Create the query data
  queryData := map[string]interface{}{
    "": map[string]interface{}{}, // Nested empty map
  }

  // Marshal the query data into JSON
  queryDataJSON, err := json.Marshal(queryData)
  if err != nil {
    log.Fatalf("Failed to marshal query data: %v", err)
  }

  // Initialize the request
  req := &wasm.QuerySmartContractStateRequest{
  Address:   contractAddress,
  QueryData: queryDataJSON,
  }

  // Call the SmartContractState method
  response, err := querier.Wasm.SmartContractState(ctx, req)
  if err != nil {
    log.Fatalf("Failed to get smart contract state: %v", err)
  }
  fmt.Printf("response: %v\n", response)
```

### Querying Token Factory Denom Information

You can query the denomination information of a token factory using the following example:

```go
  reqDenom := &tokenfactory.QueryDenomInfoRequest{
    Denom: "tf/<address>/<denom>",
  }

  denomInfo, err := querier.TokenFactory.DenomInfo(ctx, reqDenom)
  if err != nil {
    log.Fatalf("Failed to query denom info: %v", err)
  }
  fmt.Printf("denomInfo: %v\n", denomInfo)
 ```

## Setting up Nibiru SDK Keyring

To interact with the Nibiru blockchain, you need to set up the Nibiru SDK and keyring. You also need to specify the chain name and the rpc endpoint, please refer to the [Networks docs](../networks/README.md) to understand which nibiru chain you are aiming to connect to. Here's an example for `nibiru-testnet-1`:

```go
  nibiruSdk, err := gosdk.NewNibiruSdk("nibiru-testnet-1", grpcConn, "https:rpc.testnet-1.nibiru.fi")
  if err != nil {
    log.Fatalf("Failed to create nibiru sdk: %v", err)
  }

  // Setup keyring
  kring := gosdk.NewKeyring()
  keyName := "local"
  mnemonic := "" // your wallet mnemonic

  addr, err := gosdk.AddSignerToKeyringSecp256k1(kring, mnemonic, keyName)
  if err != nil {
    log.Fatalf("Failed to import wallet: %v", err)
  }

  // Define the sender address (from)
  fromAddress := sdk.AccAddress(addr)
```

### Executing a Smart Contract

Here is an example of how to execute a smart contract:

```go
  contractAddress := "" // some contract address
  
  // Create the execute message data
  executeMsg := map[string]interface{}{
    "": map[string]interface{}{},
  }

  // Marshal the execute message into JSON
  executeMsgJSON, err := json.Marshal(executeMsg)
  if err != nil {
    log.Fatalf("Failed to marshal execute message: %v", err)
  }

  // Create the MsgExecuteContract message
  msg := &wasm.MsgExecuteContract{
    Sender:   fromAddress.String(),
    Contract: contractAddress,
    Msg:      executeMsgJSON,
    Funds:    nil,
  } // nil for funds if no funds are sent

  // Broadcast the message
  res, err := nibiruSdk.BroadcastMsgs(fromAddress, msg)
  if err != nil {
    log.Fatalf("Failed to broadcast message: %v", err)
  }

  // Process the response
  fmt.Printf("Broadcast result: %v\n", res)
```

This documentation now provides a comprehensive guide on installing, setting up, and using the Gonibi SDK to interact with the Nibiru blockchain, along with examples of querying smart contract states and executing smart contracts.

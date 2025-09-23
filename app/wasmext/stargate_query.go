package wasmext

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	devgas "github.com/NibiruChain/nibiru/v2/x/devgas/v1/types"
	epochs "github.com/NibiruChain/nibiru/v2/x/epochs/types"
	inflation "github.com/NibiruChain/nibiru/v2/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/v2/x/oracle/types"
	sudotypes "github.com/NibiruChain/nibiru/v2/x/sudo/types"
	tokenfactory "github.com/NibiruChain/nibiru/v2/x/tokenfactory/types"

	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
	gov "github.com/cosmos/cosmos-sdk/x/gov/types/v1"
	"github.com/cosmos/gogoproto/proto"

	ibctransfertypes "github.com/cosmos/ibc-go/v8/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v8/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v8/modules/core/03-connection/types"
)

/*
WasmAcceptedStargateQueries: Specifies which `QueryRequest::Stargate` types
can be sent to the application.

### On Stargate Queries:

A Stargate query is encoded the same way as abci_query, with path and protobuf
encoded request data. The format is defined in
[ADR-21](https://github.com/cosmos/cosmos-sdk/blob/master/docs/architecture/adr-021-protobuf-query-encoding.md).
- The response is protobuf encoded data directly without a JSON response wrapper.
The caller is responsible for compiling the proper protobuf definitions for both
requests and responses.

	```rust
	enum QueryRequest {
	  Stargate {
		/// this is the fully qualified service path used for routing,
		/// eg. custom/cosmos_sdk.x.bank.v1.Query/QueryBalance
		path: String,
		/// this is the expected protobuf message type (not any), binary encoded
		data: Binary,
	  },
	  // ...
	}
	```

### Relationship with Protobuf Message:

A protobuf message with type URL "/cosmos.bank.v1beta1.QueryBalanceResponse"
communicates a lot of information. From this type URL, we know:
  - The protobuf message has package "cosmos.bank.v1beta1"
  - The protobuf message has name "QueryBalanceResponse"

That is, a type URL is of the form "/[PB_MSG.PACKAGE]/[PB_MSG.NAME]"

The `QueryRequest::Stargate.path` is defined based on method name of the gRPC
service description, not the type URL. In this example:
  - The service name is "cosmos.bank.v1beta1.Query"
  - The method name for this request on that service is "Balance"

This results in the expected `Stargate.path` of "/[SERVICE_NAME]/[METHOD]".
By convention, the gRPC query service corresponding to a package is always
"[PB_MSG.PACKAGE].Query".

Given only the `PB_MSG.PACKAGE` and the `PB_MSG.NAME` of either the query
request or response, we should know the `QueryRequest::Stargate.path`
deterministically.
*/
func WasmAcceptedStargateQueries() wasmkeeper.AcceptedQueries {
	return wasmkeeper.AcceptedQueries{
		// ibc
		"/ibc.core.client.v1.Query/ClientState":                  func() proto.Message { return &ibcclienttypes.QueryClientStateResponse{} },
		"/ibc.core.client.v1.Query/ConsensusState":               func() proto.Message { return &ibcclienttypes.QueryConsensusStateResponse{} },
		"/ibc.core.connection.v1.Query/Connection":               func() proto.Message { return &ibcconnectiontypes.QueryConnectionResponse{} },
		"/ibc.core.connection.v1.Query/Connections":              func() proto.Message { return &ibcconnectiontypes.QueryConnectionsResponse{} },
		"/ibc.core.connection.v1.Query/ClientConnections":        func() proto.Message { return &ibcconnectiontypes.QueryClientConnectionsResponse{} },
		"/ibc.core.connection.v1.Query/ConnectionConsensusState": func() proto.Message { return &ibcconnectiontypes.QueryConnectionConsensusStateResponse{} },
		"/ibc.core.connection.v1.Query/ConnectionParams":         func() proto.Message { return &ibcconnectiontypes.QueryConnectionParamsResponse{} },

		// ibc transfer
		"/ibc.applications.transfer.v1.Query/DenomTrace":          func() proto.Message { return &ibctransfertypes.QueryDenomTraceResponse{} },
		"/ibc.applications.transfer.v1.Query/Params":              func() proto.Message { return &ibctransfertypes.QueryParamsResponse{} },
		"/ibc.applications.transfer.v1.Query/DenomHash":           func() proto.Message { return &ibctransfertypes.QueryDenomHashResponse{} },
		"/ibc.applications.transfer.v1.Query/EscrowAddress":       func() proto.Message { return &ibctransfertypes.QueryEscrowAddressResponse{} },
		"/ibc.applications.transfer.v1.Query/TotalEscrowForDenom": func() proto.Message { return &ibctransfertypes.QueryTotalEscrowForDenomResponse{} },

		// cosmos auth
		"/cosmos.auth.v1beta1.Query/Account": func() proto.Message { return &auth.QueryAccountResponse{} },
		"/cosmos.auth.v1beta1.Query/Params":  func() proto.Message { return &auth.QueryParamsResponse{} },

		// cosmos bank
		"/cosmos.bank.v1beta1.Query/Balance":       func() proto.Message { return &bank.QueryBalanceResponse{} },
		"/cosmos.bank.v1beta1.Query/DenomMetadata": func() proto.Message { return &bank.QueryDenomMetadataResponse{} },
		"/cosmos.bank.v1beta1.Query/Params":        func() proto.Message { return &bank.QueryParamsResponse{} },
		"/cosmos.bank.v1beta1.Query/SupplyOf":      func() proto.Message { return &bank.QuerySupplyOfResponse{} },
		"/cosmos.bank.v1beta1.Query/AllBalances":   func() proto.Message { return &bank.QueryAllBalancesResponse{} },

		// cosmos gov
		"/cosmos.gov.v1.Query/Proposal": func() proto.Message { return &gov.QueryProposalResponse{} },
		"/cosmos.gov.v1.Query/Params":   func() proto.Message { return &gov.QueryParamsResponse{} },
		"/cosmos.gov.v1.Query/Vote":     func() proto.Message { return &gov.QueryVoteResponse{} },

		// nibiru tokenfactory
		"/nibiru.tokenfactory.v1.Query/Denoms":    func() proto.Message { return &tokenfactory.QueryDenomsResponse{} },
		"/nibiru.tokenfactory.v1.Query/Params":    func() proto.Message { return &tokenfactory.QueryParamsResponse{} },
		"/nibiru.tokenfactory.v1.Query/DenomInfo": func() proto.Message { return &tokenfactory.QueryDenomInfoResponse{} },

		// nibiru epochs
		"/nibiru.epochs.v1.Query/EpochInfos":   func() proto.Message { return &epochs.QueryEpochInfosResponse{} },
		"/nibiru.epochs.v1.Query/CurrentEpoch": func() proto.Message { return &epochs.QueryCurrentEpochResponse{} },

		// nibiru inflation
		"/nibiru.inflation.v1.Query/Period":             func() proto.Message { return &inflation.QueryPeriodResponse{} },
		"/nibiru.inflation.v1.Query/EpochMintProvision": func() proto.Message { return &inflation.QueryEpochMintProvisionResponse{} },
		"/nibiru.inflation.v1.Query/SkippedEpochs":      func() proto.Message { return &inflation.QuerySkippedEpochsResponse{} },
		"/nibiru.inflation.v1.Query/CirculatingSupply":  func() proto.Message { return &inflation.QueryCirculatingSupplyResponse{} },
		"/nibiru.inflation.v1.Query/InflationRate":      func() proto.Message { return &inflation.QueryInflationRateResponse{} },
		"/nibiru.inflation.v1.Query/Params":             func() proto.Message { return &inflation.QueryParamsResponse{} },

		// nibiru oracle
		"/nibiru.oracle.v1.Query/ExchangeRate":      func() proto.Message { return &oracle.QueryExchangeRateResponse{} },
		"/nibiru.oracle.v1.Query/ExchangeRateTwap":  func() proto.Message { return &oracle.QueryExchangeRateResponse{} },
		"/nibiru.oracle.v1.Query/ExchangeRates":     func() proto.Message { return &oracle.QueryExchangeRatesResponse{} },
		"/nibiru.oracle.v1.Query/Actives":           func() proto.Message { return &oracle.QueryActivesResponse{} },
		"/nibiru.oracle.v1.Query/VoteTargets":       func() proto.Message { return &oracle.QueryVoteTargetsResponse{} },
		"/nibiru.oracle.v1.Query/FeederDelegation":  func() proto.Message { return &oracle.QueryFeederDelegationResponse{} },
		"/nibiru.oracle.v1.Query/MissCounter":       func() proto.Message { return &oracle.QueryMissCounterResponse{} },
		"/nibiru.oracle.v1.Query/AggregatePrevote":  func() proto.Message { return &oracle.QueryAggregatePrevoteResponse{} },
		"/nibiru.oracle.v1.Query/AggregatePrevotes": func() proto.Message { return &oracle.QueryAggregatePrevotesResponse{} },
		"/nibiru.oracle.v1.Query/AggregateVote":     func() proto.Message { return &oracle.QueryAggregateVoteResponse{} },
		"/nibiru.oracle.v1.Query/AggregateVotes":    func() proto.Message { return &oracle.QueryAggregateVotesResponse{} },
		"/nibiru.oracle.v1.Query/Params":            func() proto.Message { return &oracle.QueryParamsResponse{} },

		// nibiru sudo
		"/nibiru.sudo.v1.Query/QuerySudoers": func() proto.Message { return &sudotypes.QuerySudoersResponse{} },

		// nibiru devgas
		"/nibiru.devgas.v1.Query/FeeShares":             func() proto.Message { return &devgas.QueryFeeSharesResponse{} },
		"/nibiru.devgas.v1.Query/FeeShare":              func() proto.Message { return &devgas.QueryFeeShareResponse{} },
		"/nibiru.devgas.v1.Query/Params":                func() proto.Message { return &devgas.QueryParamsResponse{} },
		"/nibiru.devgas.v1.Query/FeeSharesByWithdrawer": func() proto.Message { return &devgas.QueryFeeSharesByWithdrawerResponse{} },
	}
}

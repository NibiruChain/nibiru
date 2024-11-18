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

	ibctransfertypes "github.com/cosmos/ibc-go/v7/modules/apps/transfer/types"
	ibcclienttypes "github.com/cosmos/ibc-go/v7/modules/core/02-client/types"
	ibcconnectiontypes "github.com/cosmos/ibc-go/v7/modules/core/03-connection/types"
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
func WasmAcceptedStargateQueries() wasmkeeper.AcceptedStargateQueries {
	return wasmkeeper.AcceptedStargateQueries{
		// ibc
		"/ibc.core.client.v1.Query/ClientState":                  &ibcclienttypes.QueryClientStateResponse{},
		"/ibc.core.client.v1.Query/ConsensusState":               &ibcclienttypes.QueryConsensusStateResponse{},
		"/ibc.core.connection.v1.Query/Connection":               &ibcconnectiontypes.QueryConnectionResponse{},
		"/ibc.core.connection.v1.Query/Connections":              &ibcconnectiontypes.QueryConnectionsResponse{},
		"/ibc.core.connection.v1.Query/ClientConnections":        &ibcconnectiontypes.QueryClientConnectionsResponse{},
		"/ibc.core.connection.v1.Query/ConnectionConsensusState": &ibcconnectiontypes.QueryConnectionConsensusStateResponse{},
		"/ibc.core.connection.v1.Query/ConnectionParams":         &ibcconnectiontypes.QueryConnectionParamsResponse{},

		// ibc transfer
		"/ibc.applications.transfer.v1.Query/DenomTrace":          &ibctransfertypes.QueryDenomTraceResponse{},
		"/ibc.applications.transfer.v1.Query/Params":              &ibctransfertypes.QueryParamsResponse{},
		"/ibc.applications.transfer.v1.Query/DenomHash":           &ibctransfertypes.QueryDenomHashResponse{},
		"/ibc.applications.transfer.v1.Query/EscrowAddress":       &ibctransfertypes.QueryEscrowAddressResponse{},
		"/ibc.applications.transfer.v1.Query/TotalEscrowForDenom": &ibctransfertypes.QueryTotalEscrowForDenomResponse{},

		// cosmos auth
		"/cosmos.auth.v1beta1.Query/Account": new(auth.QueryAccountResponse),
		"/cosmos.auth.v1beta1.Query/Params":  new(auth.QueryParamsResponse),

		// cosmos bank
		"/cosmos.bank.v1beta1.Query/Balance":       new(bank.QueryBalanceResponse),
		"/cosmos.bank.v1beta1.Query/DenomMetadata": new(bank.QueryDenomMetadataResponse),
		"/cosmos.bank.v1beta1.Query/Params":        new(bank.QueryParamsResponse),
		"/cosmos.bank.v1beta1.Query/SupplyOf":      new(bank.QuerySupplyOfResponse),
		"/cosmos.bank.v1beta1.Query/AllBalances":   new(bank.QueryAllBalancesResponse),

		// cosmos gov
		"/cosmos.gov.v1.Query/Proposal": new(gov.QueryProposalResponse),
		"/cosmos.gov.v1.Query/Params":   new(gov.QueryParamsResponse),
		"/cosmos.gov.v1.Query/Vote":     new(gov.QueryVoteResponse),

		// nibiru tokenfactory
		"/nibiru.tokenfactory.v1.Query/Denoms":    new(tokenfactory.QueryDenomsResponse),
		"/nibiru.tokenfactory.v1.Query/Params":    new(tokenfactory.QueryParamsResponse),
		"/nibiru.tokenfactory.v1.Query/DenomInfo": new(tokenfactory.QueryDenomInfoResponse),

		// nibiru epochs
		"/nibiru.epochs.v1.Query/EpochInfos":   new(epochs.QueryEpochInfosResponse),
		"/nibiru.epochs.v1.Query/CurrentEpoch": new(epochs.QueryCurrentEpochResponse),

		// nibiru inflation
		"/nibiru.inflation.v1.Query/Period":             new(inflation.QueryPeriodResponse),
		"/nibiru.inflation.v1.Query/EpochMintProvision": new(inflation.QueryEpochMintProvisionResponse),
		"/nibiru.inflation.v1.Query/SkippedEpochs":      new(inflation.QuerySkippedEpochsResponse),
		"/nibiru.inflation.v1.Query/CirculatingSupply":  new(inflation.QueryCirculatingSupplyResponse),
		"/nibiru.inflation.v1.Query/InflationRate":      new(inflation.QueryInflationRateResponse),
		"/nibiru.inflation.v1.Query/Params":             new(inflation.QueryParamsResponse),

		// nibiru oracle
		"/nibiru.oracle.v1.Query/ExchangeRate":      new(oracle.QueryExchangeRateResponse),
		"/nibiru.oracle.v1.Query/DatedExchangeRate": new(oracle.QueryDatedExchangeRateResponse),
		"/nibiru.oracle.v1.Query/ExchangeRateTwap":  new(oracle.QueryExchangeRateResponse),
		"/nibiru.oracle.v1.Query/ExchangeRates":     new(oracle.QueryExchangeRatesResponse),
		"/nibiru.oracle.v1.Query/Actives":           new(oracle.QueryActivesResponse),
		"/nibiru.oracle.v1.Query/VoteTargets":       new(oracle.QueryVoteTargetsResponse),
		"/nibiru.oracle.v1.Query/FeederDelegation":  new(oracle.QueryFeederDelegationResponse),
		"/nibiru.oracle.v1.Query/MissCounter":       new(oracle.QueryMissCounterResponse),
		"/nibiru.oracle.v1.Query/AggregatePrevote":  new(oracle.QueryAggregatePrevoteResponse),
		"/nibiru.oracle.v1.Query/AggregatePrevotes": new(oracle.QueryAggregatePrevotesResponse),
		"/nibiru.oracle.v1.Query/AggregateVote":     new(oracle.QueryAggregateVoteResponse),
		"/nibiru.oracle.v1.Query/AggregateVotes":    new(oracle.QueryAggregateVotesResponse),
		"/nibiru.oracle.v1.Query/Params":            new(oracle.QueryParamsResponse),

		// nibiru sudo
		"/nibiru.sudo.v1.Query/QuerySudoers": new(sudotypes.QuerySudoersResponse),

		// nibiru devgas
		"/nibiru.devgas.v1.Query/FeeShares":             new(devgas.QueryFeeSharesResponse),
		"/nibiru.devgas.v1.Query/FeeShare":              new(devgas.QueryFeeShareResponse),
		"/nibiru.devgas.v1.Query/Params":                new(devgas.QueryParamsResponse),
		"/nibiru.devgas.v1.Query/FeeSharesByWithdrawer": new(devgas.QueryFeeSharesByWithdrawerResponse),
	}
}

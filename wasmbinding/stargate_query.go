package wasmbinding

import (
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	devgas "github.com/NibiruChain/nibiru/x/devgas/v1/types"
	epochs "github.com/NibiruChain/nibiru/x/epochs/types"
	inflation "github.com/NibiruChain/nibiru/x/inflation/types"
	oracle "github.com/NibiruChain/nibiru/x/oracle/types"
	sudotypes "github.com/NibiruChain/nibiru/x/sudo/types"
	tokenfactory "github.com/NibiruChain/nibiru/x/tokenfactory/types"

	auth "github.com/cosmos/cosmos-sdk/x/auth/types"
	bank "github.com/cosmos/cosmos-sdk/x/bank/types"
)

func WasmAcceptedStargateQueries() wasmkeeper.AcceptedStargateQueries {
	return wasmkeeper.AcceptedStargateQueries{
		// auth
		"/cosmos.auth.v1beta1.Query/Account": new(auth.QueryAccountResponse),
		"/cosmos.auth.v1beta1.Query/Params":  new(auth.QueryParamsResponse),

		// bank
		"/cosmos.bank.v1beta1.Query/Balance":       new(bank.QueryBalanceResponse),
		"/cosmos.bank.v1beta1.Query/DenomMetadata": new(bank.QueryDenomMetadataResponse),
		"/cosmos.bank.v1beta1.Query/Params":        new(bank.QueryParamsResponse),
		"/cosmos.bank.v1beta1.Query/SupplyOf":      new(bank.QuerySupplyOfResponse),

		// nibiru - tokenfactory
		"/nibiru.tokenfactory.v1.Query/Denoms":    new(tokenfactory.QueryDenomsResponse),
		"/nibiru.tokenfactory.v1.Query/Params":    new(tokenfactory.QueryParamsResponse),
		"/nibiru.tokenfactory.v1.Query/DenomInfo": new(tokenfactory.QueryDenomInfoResponse),

		// nibiru - epochs
		"/nibiru.epochs.v1.Query/EpochInfos":   new(epochs.QueryEpochsInfoResponse),
		"/nibiru.epochs.v1.Query/CurrentEpoch": new(epochs.QueryCurrentEpochResponse),

		// nibiru - inflation
		"/nibiru.inflation.v1.Query/Period":             new(inflation.QueryPeriodResponse),
		"/nibiru.inflation.v1.Query/EpochMintProvision": new(inflation.QueryEpochMintProvisionResponse),
		"/nibiru.inflation.v1.Query/SkippedEpochs":      new(inflation.QuerySkippedEpochsResponse),
		"/nibiru.inflation.v1.Query/CirculatingSupply":  new(inflation.QueryCirculatingSupplyResponse),
		"/nibiru.inflation.v1.Query/InflationRate":      new(inflation.QueryInflationRateResponse),
		"/nibiru.inflation.v1.Query/Params":             new(inflation.QueryParamsResponse),

		// nibiru - oracle
		"/nibiru.oracle.v1.Query/ExchangeRate":      new(oracle.QueryExchangeRateResponse),
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

		// nibiru - sudo
		"/nibiru.sudo.v1.Query/QuerySudoers": new(sudotypes.QuerySudoersResponse),

		// nibiru - devgas
		"/nibiru.devgas.v1.Query/FeeShares":             new(devgas.QueryFeeSharesResponse),
		"/nibiru.devgas.v1.Query/FeeShare":              new(devgas.QueryFeeShareResponse),
		"/nibiru.devgas.v1.Query/Params":                new(devgas.QueryParamsResponse),
		"/nibiru.devgas.v1.Query/FeeSharesByWithdrawer": new(devgas.QueryFeeSharesByWithdrawerResponse),

		// TODO for post v1
		// nibiru - perp

		// TODO for post v1
		// nibiru - spot
		// "/nibiru.tokenfactory.v1.Query/Params":      &tokenfactory.QueryParamsResponse{},
		// "/nibiru.tokenfactory.v1.Query/DenomInfo":      &tokenfactory.QueryDenomInfoResponse{},

	}
}

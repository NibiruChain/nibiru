package evmtrader

import (
	"context"
	"time"
)

type status24hMetrics struct {
	positionsOpened24h int
	positionsClosed24h int
	volumeUSD          float64
	realizedPnL        float64
	hasVolumePnL       bool
	graphQLError       string
}

// compute24hMetrics loads 24h open/close/volume/PnL from sai-keeper GraphQL only.
// Failed txs come from in-memory bot tracking, not GraphQL.
func compute24hMetrics(
	ctx context.Context,
	now time.Time,
	gql *KeeperGraphQLClient,
	traderAddr string,
) status24hMetrics {
	out := status24hMetrics{}

	if gql == nil {
		out.graphQLError = "graphql client not configured"
		return out
	}
	if ctx == nil || ctx.Err() != nil {
		out.graphQLError = "context unavailable for graphql"
		return out
	}
	if traderAddr == "" {
		out.graphQLError = "empty trader address"
		return out
	}

	activity, err := gql.Compute24hActivity(ctx, traderAddr, now)
	if err != nil {
		out.graphQLError = err.Error()
		return out
	}

	out.positionsOpened24h = activity.PositionsOpened24h
	out.positionsClosed24h = activity.PositionsClosed24h
	if activity.HasVolumePnL {
		out.volumeUSD = activity.VolumeUSD
		out.realizedPnL = activity.RealizedPnLUSD
		out.hasVolumePnL = true
	}
	return out
}

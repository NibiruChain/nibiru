package precompile

import (
	"testing"

	xoracle "github.com/NibiruChain/nibiru/v2/x/oracle"
)

func TestXOracleAdapterLegacyExchangeRateRespTimestamps(t *testing.T) {
	updateTimeSeconds := uint64(1_700_000_123)
	resp := xoracle.XOracleAdapterLegacyExchangeRateResp{
		UpdateTimeSeconds: &updateTimeSeconds,
	}

	if got := adapterUpdateTimeSeconds(resp); got != updateTimeSeconds {
		t.Fatalf("adapterUpdateTimeSeconds() = %d, want %d", got, updateTimeSeconds)
	}
	if got, want := adapterBlockTimeMs(resp), updateTimeSeconds*1000; got != want {
		t.Fatalf("adapterBlockTimeMs() = %d, want %d", got, want)
	}
}

func TestXOracleAdapterLegacyExchangeRateRespMissingTimestampIsUnknown(t *testing.T) {
	resp := xoracle.XOracleAdapterLegacyExchangeRateResp{}

	if got := adapterUpdateTimeSeconds(resp); got != 0 {
		t.Fatalf("adapterUpdateTimeSeconds() = %d, want unknown timestamp 0", got)
	}
	if got := adapterBlockTimeMs(resp); got != 0 {
		t.Fatalf("adapterBlockTimeMs() = %d, want unknown timestamp 0", got)
	}
}

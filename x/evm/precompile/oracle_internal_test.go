package precompile

import "testing"

func TestXOracleAdapterLegacyExchangeRateRespTimestamps(t *testing.T) {
	updateTimeSeconds := uint64(1_700_000_123)
	resp := xOracleAdapterLegacyExchangeRateResp{
		UpdateTimeSeconds: &updateTimeSeconds,
	}

	if got := resp.updateTimeSeconds(); got != updateTimeSeconds {
		t.Fatalf("updateTimeSeconds() = %d, want %d", got, updateTimeSeconds)
	}
	if got, want := resp.blockTimeMs(), updateTimeSeconds*1000; got != want {
		t.Fatalf("blockTimeMs() = %d, want %d", got, want)
	}
}

func TestXOracleAdapterLegacyExchangeRateRespMissingTimestampIsUnknown(t *testing.T) {
	resp := xOracleAdapterLegacyExchangeRateResp{}

	if got := resp.updateTimeSeconds(); got != 0 {
		t.Fatalf("updateTimeSeconds() = %d, want unknown timestamp 0", got)
	}
	if got := resp.blockTimeMs(); got != 0 {
		t.Fatalf("blockTimeMs() = %d, want unknown timestamp 0", got)
	}
}

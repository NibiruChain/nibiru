package ewma

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/stretchr/testify/require"
)

func TestSimpleEWMA(t *testing.T) {
	file, err := os.Open("testdata/ewma.csv")
	require.NoError(t, err)
	defer file.Close()

	// discard the header
	reader := csv.NewReader(file)
	_, err = reader.Read()
	require.NoError(t, err)

	ewma := NewMovingAverage(sdkmath.LegacyMustNewDecFromStr("240"))

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		ewma.Add(sdkmath.LegacyMustNewDecFromStr(record[1]))
		require.Equal(
			t,
			sdkmath.LegacyMustNewDecFromStr(record[2]).Mul(sdkmath.LegacyMustNewDecFromStr("100000")).TruncateInt(),
			ewma.Value().Mul(sdkmath.LegacyMustNewDecFromStr("100000")).TruncateInt(),
			fmt.Sprintf("value in position %s: %s should be equal to %s", record[0], ewma.Value().Mul(sdkmath.LegacyMustNewDecFromStr("1000000")).TruncateInt().String(), record[2]),
		)
	}
}

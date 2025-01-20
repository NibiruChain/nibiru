package ewma

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"testing"

	"cosmossdk.io/math"
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

	ewma := NewMovingAverage(math.LegacyMustNewDecFromStr("240"))

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		ewma.Add(math.LegacyMustNewDecFromStr(record[1]))
		require.Equal(
			t,
			math.LegacyMustNewDecFromStr(record[2]).Mul(math.LegacyMustNewDecFromStr("100000")).TruncateInt(),
			ewma.Value().Mul(math.LegacyMustNewDecFromStr("100000")).TruncateInt(),
			fmt.Sprintf("value in position %s: %s should be equal to %s", record[0], ewma.Value().Mul(math.LegacyMustNewDecFromStr("1000000")).TruncateInt().String(), record[2]),
		)
	}
}

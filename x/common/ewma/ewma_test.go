package ewma

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
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

	ewma := NewMovingAverage(sdk.MustNewDecFromStr("240"))

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)

		ewma.Add(sdk.MustNewDecFromStr(record[1]))
		require.Equal(
			t,
			sdk.MustNewDecFromStr(record[2]).Mul(sdk.MustNewDecFromStr("100000")).TruncateInt(),
			ewma.Value().Mul(sdk.MustNewDecFromStr("100000")).TruncateInt(),
			fmt.Sprintf("value in position %s: %s should be equal to %s", record[0], ewma.Value().Mul(sdk.MustNewDecFromStr("1000000")).TruncateInt().String(), record[2]),
		)
	}
}

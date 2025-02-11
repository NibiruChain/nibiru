package gosdk_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/gosdk"
)

const LOCALNET_VALIDATOR_MNEMONIC = "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"

func TestCreateSigner(t *testing.T) {
	testCases := []struct {
		testName  string
		mnemonic  string
		wantAddr  string
		expectErr bool
	}{
		{
			testName:  "bad input",
			mnemonic:  "not a mnemonic",
			expectErr: true,
		},
		{
			testName:  "good input (localnet genesis)",
			mnemonic:  LOCALNET_VALIDATOR_MNEMONIC,
			wantAddr:  "nibi1zaavvzxez0elundtn32qnk9lkm8kmcsz44g7xl",
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			kring := gosdk.NewKeyring()
			keyName := ""

			gotAddr, err := gosdk.AddSignerToKeyringSecp256k1(kring, tc.mnemonic, keyName)

			if tc.expectErr {
				require.Error(t, err)
				return
			}

			assert.EqualValues(t, gotAddr.String(), tc.wantAddr)
			require.NoError(t, err)
		})
	}
}

func TestKeyring(t *testing.T) {
	require.NotPanics(t, func() {
		_ = gosdk.NewKeyring()
	})
}

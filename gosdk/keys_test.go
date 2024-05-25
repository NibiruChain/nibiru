package gosdk_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/gosdk"
)

const LOCALNET_VALIDATOR_MNEMONIC = "guard cream sadness conduct invite crumble clock pudding hole grit liar hotel maid produce squeeze return argue turtle know drive eight casino maze host"

func TestCreateSigner(t *testing.T) {
	testCases := []struct {
		testName  string
		mnemonic  string
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
			expectErr: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			kring := gosdk.NewKeyring()
			keyName := ""
			signer, privKey, err := gosdk.CreateSigner(tc.mnemonic, kring, keyName)
			if tc.expectErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.NotNil(t, signer.PubKey)

			err = gosdk.AddSignerToKeyring(kring, privKey, privKey.PubKey().String())
			require.NoError(t, err)
		})
	}
}

func TestKeyring(t *testing.T) {
	require.NotPanics(t, func() {
		_ = gosdk.NewKeyring()
	})
}

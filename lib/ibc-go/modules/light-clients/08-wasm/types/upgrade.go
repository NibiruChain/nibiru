package types

import (
	sdkioerrors "cosmossdk.io/errors"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/codec"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	ibcerrors "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/errors"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
)

// VerifyUpgradeAndUpdateState, on a successful verification expects the contract to update
// the new client state, consensus state, and any other client metadata.
func (cs ClientState) VerifyUpgradeAndUpdateState(
	ctx sdk.Context,
	cdc codec.BinaryCodec,
	clientStore sdk.KVStore,
	upgradedClient exported.ClientState,
	upgradedConsState exported.ConsensusState,
	proofUpgradeClient,
	proofUpgradeConsState []byte,
) error {
	wasmUpgradeClientState, ok := upgradedClient.(*ClientState)
	if !ok {
		return sdkioerrors.Wrapf(clienttypes.ErrInvalidClient, "upgraded client state must be wasm light client state. expected %T, got: %T",
			&ClientState{}, wasmUpgradeClientState)
	}

	wasmUpgradeConsState, ok := upgradedConsState.(*ConsensusState)
	if !ok {
		return sdkioerrors.Wrapf(clienttypes.ErrInvalidConsensus, "upgraded consensus state must be wasm light consensus state. expected %T, got: %T",
			&ConsensusState{}, wasmUpgradeConsState)
	}

	// last height of current counterparty chain must be client's latest height
	lastHeight := cs.GetLatestHeight()

	if !upgradedClient.GetLatestHeight().GT(lastHeight) {
		return sdkioerrors.Wrapf(ibcerrors.ErrInvalidHeight, "upgraded client height %s must be greater than current client height %s",
			upgradedClient.GetLatestHeight(), lastHeight)
	}

	payload := SudoMsg{
		VerifyUpgradeAndUpdateState: &VerifyUpgradeAndUpdateStateMsg{
			UpgradeClientState:         wasmUpgradeClientState.Data,
			UpgradeConsensusState:      wasmUpgradeConsState.Data,
			ProofUpgradeClient:         proofUpgradeClient,
			ProofUpgradeConsensusState: proofUpgradeConsState,
		},
	}

	_, err := wasmSudo[EmptyResult](ctx, cdc, clientStore, &cs, payload)
	return err
}

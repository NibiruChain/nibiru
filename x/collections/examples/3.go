package examples

import (
	"github.com/NibiruChain/nibiru/v2/x/collections"
	"github.com/cosmos/cosmos-sdk/codec"
	storagetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
)

// ValidatorIndexes defines the indexes for the Validator IndexedMap.
// This structure just defines and groups together the indexes associated
// with a value of the IndexedMap.
type ValidatorIndexes struct {
	// ConsensusAddress defines an Index of the validator structure which allows us
	// to get validators by their ConsensusAddress.
	ConsensusAddress collections.MultiIndex[sdk.ConsAddress, sdk.ValAddress, types.Validator]
}

// IndexerList implements the collections.IndexersProvider interface.
// It consists on simply returning the list of the indexes (we have only one ConsensusAddress)
func (v ValidatorIndexes) IndexerList() []collections.Indexer[sdk.ValAddress, types.Validator] {
	return []collections.Indexer[sdk.ValAddress, types.Validator]{v.ConsensusAddress}
}

type StakingKeeper2 struct {
	Validators collections.IndexedMap[sdk.ValAddress, types.Validator, ValidatorIndexes]
}

func NewStakingKeeper2(sk storagetypes.StoreKey, cdc codec.BinaryCodec) *StakingKeeper2 {
	return &StakingKeeper2{
		Validators: collections.NewIndexedMap(
			sk, 0,
			collections.ValAddressKeyEncoder,                    // defining how we encode the primary key
			collections.ProtoValueEncoder[types.Validator](cdc), // defining how we enode the types.Validator object
			ValidatorIndexes{
				ConsensusAddress: collections.NewMultiIndex(
					sk,
					1,                                 // NOTE Indexes namespace needs to be unique across every other collections object in the module.
					collections.ConsAddressKeyEncoder, // we define how we encode the index key.
					collections.ValAddressKeyEncoder,  // we define how we encode the primary key (again).
					func(v types.Validator) sdk.ConsAddress {
						// this is the function that given the object it returns the secondary key
						// aka we return the sdk.ConsAddress of the validator because we want to map
						// validators to their consensus address
						consAddr, err := v.GetConsAddr()
						if err != nil {
							panic(err)
						}
						return consAddr
					},
				),
			},
		),
	}
}

func (k StakingKeeper2) CreateValidator(ctx sdk.Context, val types.Validator) {
	k.Validators.Insert(ctx, val.GetOperator(), val)
}

func (k StakingKeeper2) GetValidatorsByConsAddress(ctx sdk.Context, consAddr sdk.ConsAddress) []types.Validator {
	pks := k.Validators.Indexes.ConsensusAddress.ExactMatch(ctx, consAddr)
	return k.Validators.Collect(ctx, pks)
}

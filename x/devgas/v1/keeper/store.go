package keeper

import (
	"cosmossdk.io/collections"
	collindexes "cosmossdk.io/collections/indexes"
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/codec"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/runtime"

	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

type DevGasIndexes struct {
	// Deployer MultiIndex:
	//  - indexing key (IK): deployer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Deployer collindexes.Multi[string, string, devgastypes.FeeShare]

	// Withdrawer MultiIndex:
	//  - indexing key (IK): withdrawer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Withdrawer collindexes.Multi[string, string, devgastypes.FeeShare]
}

func (idxs DevGasIndexes) IndexesList() []collections.Index[string, devgastypes.FeeShare] {
	return []collections.Index[string, devgastypes.FeeShare]{
		&idxs.Deployer, &idxs.Withdrawer,
	}
}

func NewDevGasStore(
	storeKey *storetypes.KVStoreKey, cdc sdkcodec.BinaryCodec,
) collections.IndexedMap[string, devgastypes.FeeShare, DevGasIndexes] {
	storeService := runtime.NewKVStoreService(storeKey)
	sb := collections.NewSchemaBuilder(storeService)

	primaryKeyEncoder := collections.StringKey
	valueEncoder := codec.CollValue[devgastypes.FeeShare](cdc)

	var namespace = devgastypes.KeyPrefixFeeShare
	var namespaceDeployerIdx = devgastypes.KeyPrefixDeployer
	var namespaceWithdrawerIdx = devgastypes.KeyPrefixWithdrawer

	return *collections.NewIndexedMap[string, devgastypes.FeeShare](
		sb,
		collections.NewPrefix(int(namespace)),
		storeKey.String(),
		primaryKeyEncoder,
		valueEncoder,
		DevGasIndexes{
			Deployer: *collindexes.NewMulti[string, string, devgastypes.FeeShare](
				sb,
				collections.NewPrefix(int(namespaceDeployerIdx)),
				storeKey.String(),
				collections.StringKey, // index key (IK)
				collections.StringKey, // primary key (PK)
				func(pk string, v devgastypes.FeeShare) (string, error) { return v.DeployerAddress, nil },
			),
			Withdrawer: *collindexes.NewMulti[string, string, devgastypes.FeeShare](
				sb,
				collections.NewPrefix(int(namespaceWithdrawerIdx)),
				storeKey.String(),
				collections.StringKey, // index key (IK)
				collections.StringKey, // primary key (PK)
				func(pk string, v devgastypes.FeeShare) (string, error) { return v.WithdrawerAddress, nil },
			),
		},
	)
}

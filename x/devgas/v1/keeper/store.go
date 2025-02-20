package keeper

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/NibiruChain/collections"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"

	devgastypes "github.com/NibiruChain/nibiru/x/devgas/v1/types"
)

type DevGasIndexes struct {
	// Deployer MultiIndex:
	//  - indexing key (IK): deployer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Deployer collections.MultiIndex[string, string, devgastypes.FeeShare]

	// Withdrawer MultiIndex:
	//  - indexing key (IK): withdrawer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Withdrawer collections.MultiIndex[string, string, devgastypes.FeeShare]
}

func (idxs DevGasIndexes) IndexerList() []collections.Indexer[string, devgastypes.FeeShare] {
	return []collections.Indexer[string, devgastypes.FeeShare]{
		idxs.Deployer, idxs.Withdrawer,
	}
}

func NewDevGasStore(
	storeKey storetypes.StoreKey, cdc sdkcodec.BinaryCodec,
) collections.IndexedMap[string, devgastypes.FeeShare, DevGasIndexes] {
	primaryKeyEncoder := collections.StringKeyEncoder
	valueEncoder := collections.ProtoValueEncoder[devgastypes.FeeShare](cdc)

	var namespace collections.Namespace = devgastypes.KeyPrefixFeeShare
	var namespaceDeployerIdx collections.Namespace = devgastypes.KeyPrefixDeployer
	var namespaceWithdrawerIdx collections.Namespace = devgastypes.KeyPrefixWithdrawer

	return collections.NewIndexedMap[string, devgastypes.FeeShare](
		storeKey, namespace, primaryKeyEncoder, valueEncoder,
		DevGasIndexes{
			Deployer: collections.NewMultiIndex[string, string, devgastypes.FeeShare](
				storeKey, namespaceDeployerIdx,
				collections.StringKeyEncoder, // index key (IK)
				collections.StringKeyEncoder, // primary key (PK)
				func(v devgastypes.FeeShare) string { return v.DeployerAddress },
			),
			Withdrawer: collections.NewMultiIndex[string, string, devgastypes.FeeShare](
				storeKey, namespaceWithdrawerIdx,
				collections.StringKeyEncoder, // index key (IK)
				collections.StringKeyEncoder, // primary key (PK)
				func(v devgastypes.FeeShare) string { return v.WithdrawerAddress },
			),
		},
	)
}

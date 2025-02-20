package keeper

import (
	"github.com/NibiruChain/collections"
	sdkcodec "github.com/cosmos/cosmos-sdk/codec"
	storetypes "github.com/cosmos/cosmos-sdk/store/types"

	"github.com/NibiruChain/nibiru/v2/x/devgas"
)

type DevGasIndexes struct {
	// Deployer MultiIndex:
	//  - indexing key (IK): deployer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Deployer collections.MultiIndex[string, string, devgas.FeeShare]

	// Withdrawer MultiIndex:
	//  - indexing key (IK): withdrawer address
	//  - primary key (PK): contract address
	//  - value (V): Dev gas struct
	Withdrawer collections.MultiIndex[string, string, devgas.FeeShare]
}

func (idxs DevGasIndexes) IndexerList() []collections.Indexer[string, devgas.FeeShare] {
	return []collections.Indexer[string, devgas.FeeShare]{
		idxs.Deployer, idxs.Withdrawer,
	}
}

func NewDevGasStore(
	storeKey storetypes.StoreKey, cdc sdkcodec.BinaryCodec,
) collections.IndexedMap[string, devgas.FeeShare, DevGasIndexes] {
	primaryKeyEncoder := collections.StringKeyEncoder
	valueEncoder := collections.ProtoValueEncoder[devgas.FeeShare](cdc)

	var namespace collections.Namespace = devgas.KeyPrefixFeeShare
	var namespaceDeployerIdx collections.Namespace = devgas.KeyPrefixDeployer
	var namespaceWithdrawerIdx collections.Namespace = devgas.KeyPrefixWithdrawer

	return collections.NewIndexedMap[string, devgas.FeeShare](
		storeKey, namespace, primaryKeyEncoder, valueEncoder,
		DevGasIndexes{
			Deployer: collections.NewMultiIndex[string, string, devgas.FeeShare](
				storeKey, namespaceDeployerIdx,
				collections.StringKeyEncoder, // index key (IK)
				collections.StringKeyEncoder, // primary key (PK)
				func(v devgas.FeeShare) string { return v.DeployerAddress },
			),
			Withdrawer: collections.NewMultiIndex[string, string, devgas.FeeShare](
				storeKey, namespaceWithdrawerIdx,
				collections.StringKeyEncoder, // index key (IK)
				collections.StringKeyEncoder, // primary key (PK)
				func(v devgas.FeeShare) string { return v.WithdrawerAddress },
			),
		},
	)
}

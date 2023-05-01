package sudo

import (
	"github.com/NibiruChain/collections"
	"github.com/NibiruChain/nibiru/x/common/set"
	"github.com/NibiruChain/nibiru/x/sudo/pb"
	"github.com/cosmos/cosmos-sdk/codec"
)

func SudoersValueEncoder(cdc codec.BinaryCodec) collections.ValueEncoder[pb.Sudoers] {
	return collections.ProtoValueEncoder[pb.Sudoers](cdc)
}

type PbSudoers = pb.Sudoers

// TODO test
func (sudo Sudoers) FromPbSudoers(pbSudoers pb.Sudoers) Sudoers {
	return Sudoers{
		Root:      pbSudoers.Root,
		Contracts: set.New[string](pbSudoers.Contracts...),
	}
}

// TODO test
func SudoersToPb(sudoers Sudoers) pb.Sudoers {
	return pb.Sudoers{
		Root:      sudoers.Root,
		Contracts: sudoers.Contracts.ToSlice(),
	}
}

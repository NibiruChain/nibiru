package precompile

import (
	// "github.com/ethereum/go-ethereum/accounts/abi"
	gethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	// "github.com/NibiruChain/nibiru/v2/x/evm/statedb"
)

func MakeTopicFromStr(s string) (topic gethcommon.Hash) {
	return MakeTopicFromBytes([]byte(s))
}

func MakeTopicFromBytes(bz []byte) (topic gethcommon.Hash) {
	hash := crypto.Keccak256Hash(bz)
	copy(topic[:], hash[:])
	return topic
}

// func (p *precompileWasm) foo(stateDB *statedb.StateDB) {
// 	abi.MakeTopics()
// 	// stateDB.AddLog()
// 	// p.evmKeeper.State
// }

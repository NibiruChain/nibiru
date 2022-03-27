package types

import fmt "fmt"

/*
Returns the denomination of a pool share token for a given poolId.

args:
  poolId: the pool id number

ret:
  poolDenom: the pool denomination name of the poolId
*/
func GetPoolShareDenom(poolId uint64) (poolDenom string) {
	return fmt.Sprintf("matrix/pool/%d", poolId)
}

package stablecoin_test

import (
	"reflect"
	"testing"
	"unsafe"

	"github.com/MatrixDao/matrix/x/stablecoin"
	keepertest "github.com/MatrixDao/matrix/x/stablecoin/testutil"
	"github.com/MatrixDao/matrix/x/stablecoin/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

var (
	coinType  = reflect.TypeOf(sdk.Coin{})
	coinsType = reflect.TypeOf(sdk.Coins{})
)

// Fill analyze all struct fields and slices with reflection.
// Initialize the nil and empty slices, structs, and pointers.
func Fill(x interface{}) interface{} {
	v := reflect.Indirect(reflect.ValueOf(x))
	switch v.Kind() {
	case reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			obj := v.Index(i)
			objPt := reflect.NewAt(obj.Type(), unsafe.Pointer(obj.UnsafeAddr())).Interface()
			objPt = Fill(objPt)
			obj.Set(reflect.ValueOf(objPt))
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			f := reflect.Indirect(v.Field(i))
			if !f.CanSet() {
				continue
			}
			switch f.Kind() {
			case reflect.Slice:
				f.Set(reflect.MakeSlice(f.Type(), 0, 0))
			case reflect.Struct:
				switch f.Type() {
				case coinType:
					coin := reflect.New(coinType).Interface()
					s := reflect.ValueOf(coin).Elem()
					f.Set(s)
				case coinsType:
					coins := reflect.New(coinsType).Interface()
					s := reflect.ValueOf(coins).Elem()
					f.Set(s)
				default:
					objPt := reflect.NewAt(f.Type(), unsafe.Pointer(f.UnsafeAddr())).Interface()
					s := Fill(objPt)
					f.Set(reflect.ValueOf(s))
				}
			}
		}
	}
	return reflect.Indirect(v).Interface()
}

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.StablecoinKeeper(t)
	stablecoin.InitGenesis(ctx, *k, genesisState)
	got := stablecoin.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	Fill(&genesisState)
	Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}

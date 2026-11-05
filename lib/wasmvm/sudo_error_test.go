//go:build cgo

package wasmvm

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/internal/api"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
)

func TestSudoDistinguishesContractErrorFromMissingCode(t *testing.T) {
	vm := withVM(t)
	checksum := createTestContract(t, vm, HACKATOM_TEST_CONTRACT)
	meter := api.NewMockGasMeter(TESTING_GAS_LIMIT)
	store := api.NewLookup(meter)
	goapi := api.NewMockAPI()
	querier := api.DefaultQuerier(api.MOCK_CONTRACT_ADDR, nil)
	env := api.MockEnv()
	cost := wvm.UFraction{Numerator: 1, Denominator: 1}
	_, _, err := vm.Instantiate(checksum, env, api.MockInfo("creator", nil),
		[]byte(`{"verifier":"fred","beneficiary":"bob"}`), store, *goapi,
		querier, meter, TESTING_GAS_LIMIT, cost)
	require.NoError(t, err)

	_, _, err = vm.Sudo(checksum, env, []byte(`{"unknown_operation":{}}`),
		store, *goapi, querier, meter, TESTING_GAS_LIMIT, cost)
	var contractErr *wvm.ContractError
	require.ErrorAs(t, err, &contractErr)
	require.Contains(t, contractErr.Message, "unknown")

	missing := append(Checksum(nil), checksum...)
	missing[0] ^= 0xff
	_, _, err = vm.Sudo(missing, env, []byte(`{}`), store, *goapi,
		querier, meter, TESTING_GAS_LIMIT, cost)
	require.Error(t, err)
	require.NotErrorAs(t, err, &contractErr)
}

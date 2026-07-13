package types_test

import (
	"encoding/json"
	"fmt"

	wasmvm "github.com/NibiruChain/nibiru/v2/lib/wasmvm"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"

	sdkioerrors "cosmossdk.io/errors"

	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"

	clienttypes "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/02-client/types"
	host "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/24-host"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/core/exported"
	ibctm "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/07-tendermint"
	wasmtesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/08-wasm/testing"
	"github.com/NibiruChain/nibiru/v2/lib/ibc-go/modules/light-clients/08-wasm/types"
	ibctesting "github.com/NibiruChain/nibiru/v2/lib/ibc-go/testing"
)

func (suite *TypesTestSuite) TestUpdateState() {
	mockHeight := clienttypes.NewHeight(1, 50)

	var (
		clientMsg             exported.ClientMessage
		clientStore           sdk.KVStore
		expectedClientStateBz []byte
	)

	testCases := []struct {
		name       string
		malleate   func()
		expPanic   error
		expHeights []exported.Height
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, env wvm.Env, sudoMsg []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.UpdateState)
					suite.Require().NotNil(msg.UpdateState.ClientMessage)
					suite.Require().Equal(msg.UpdateState.ClientMessage, clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader))
					suite.Require().Nil(msg.VerifyMembership)
					suite.Require().Nil(msg.VerifyNonMembership)
					suite.Require().Nil(msg.UpdateStateOnMisbehaviour)
					suite.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					suite.Require().Equal(env.Contract.Address, defaultWasmClientID)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wvm.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{},
		},
		{
			"success: update client",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wvm.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					bz := store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(bz)
					clientState := clienttypes.MustUnmarshalClientState(suite.chainA.Codec, bz).(*types.ClientState)
					clientState.LatestHeight = mockHeight
					expectedClientStateBz = clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), expectedClientStateBz)

					updateStateResp := types.UpdateStateResult{
						Heights: []clienttypes.Height{mockHeight},
					}

					resp, err := json.Marshal(updateStateResp)
					if err != nil {
						return nil, 0, err
					}

					return &wvm.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			[]exported.Height{mockHeight},
		},
		{
			"failure: clientStore prefix does not include clientID",
			func() {
				clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), ibctesting.InvalidID)
			},
			sdkioerrors.Wrap(types.ErrWasmContractCallFailed, sdkioerrors.Wrap(sdkioerrors.Wrapf(types.ErrRetrieveClientID, "prefix does not contain a valid clientID: %s", sdkioerrors.Wrapf(host.ErrInvalidID, "invalid client identifier %s", ibctesting.InvalidID)), "failed to retrieve clientID for wasm contract call").Error()),
			nil,
		},
		{
			"failure: invalid ClientMessage type",
			func() {
				// SudoCallback left nil because clientMsg is checked by 08-wasm before callbackFn is called.
				clientMsg = &ibctm.Misbehaviour{}
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*ibctm.Misbehaviour)(nil)),
			nil,
		},
		{
			"failure: callbackFn returns error",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateMsg{}, func(_ wasmvm.Checksum, _ wvm.Env, _ []byte, _ wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			sdkioerrors.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			suite.SetupWasmWithMockVM() // reset
			expectedClientStateBz = nil

			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientHeader),
			}

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)
			clientStore = suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)

			tc.malleate()

			clientState := endpoint.GetClientState()

			var heights []exported.Height
			updateState := func() {
				heights = clientState.UpdateState(suite.chainA.GetContext(), suite.chainA.Codec, clientStore, clientMsg)
			}

			if tc.expPanic == nil {
				updateState()
				suite.Require().Equal(tc.expHeights, heights)

				if expectedClientStateBz != nil {
					clientStateBz := clientStore.Get(host.ClientStateKey())
					suite.Require().Equal(expectedClientStateBz, clientStateBz)
				}
			} else {
				suite.Require().PanicsWithError(tc.expPanic.Error(), updateState)
			}
		})
	}
}

func (suite *TypesTestSuite) TestUpdateStateOnMisbehaviour() {
	mockHeight := clienttypes.NewHeight(1, 50)

	var clientMsg exported.ClientMessage

	var expectedClientStateBz []byte

	testCases := []struct {
		name               string
		malleate           func()
		panicErr           error
		updatedClientState []byte
	}{
		{
			"success: no update",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wvm.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					var msg types.SudoMsg

					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					suite.Require().NotNil(msg.UpdateStateOnMisbehaviour)
					suite.Require().NotNil(msg.UpdateStateOnMisbehaviour.ClientMessage)
					suite.Require().Nil(msg.UpdateState)
					suite.Require().Nil(msg.UpdateState)
					suite.Require().Nil(msg.VerifyMembership)
					suite.Require().Nil(msg.VerifyNonMembership)
					suite.Require().Nil(msg.VerifyUpgradeAndUpdateState)

					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wvm.Response{
						Data: resp,
					}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			nil,
		},
		{
			"success: client state updated on valid misbehaviour",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wvm.Env, sudoMsg []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					var msg types.SudoMsg
					err := json.Unmarshal(sudoMsg, &msg)
					suite.Require().NoError(err)

					// set new client state in store
					bz := store.Get(host.ClientStateKey())
					suite.Require().NotEmpty(bz)
					clientState := clienttypes.MustUnmarshalClientState(suite.chainA.App.AppCodec(), bz).(*types.ClientState)
					clientState.LatestHeight = mockHeight
					expectedClientStateBz = clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), clientState)
					store.Set(host.ClientStateKey(), expectedClientStateBz)

					resp, err := json.Marshal(types.EmptyResult{})
					if err != nil {
						return nil, 0, err
					}

					return &wvm.Response{Data: resp}, wasmtesting.DefaultGasUsed, nil
				})
			},
			nil,
			clienttypes.MustMarshalClientState(suite.chainA.App.AppCodec(), wasmtesting.CreateMockTendermintClientState(mockHeight)),
		},
		{
			"failure: invalid client message",
			func() {
				clientMsg = &ibctm.Header{}
				// we will not register the callback here because this test case does not reach the VM
			},
			fmt.Errorf("expected type %T, got %T", (*types.ClientMessage)(nil), (*ibctm.Header)(nil)),
			nil,
		},
		{
			"failure: err return from contract vm",
			func() {
				suite.mockVM.RegisterSudoCallback(types.UpdateStateOnMisbehaviourMsg{}, func(_ wasmvm.Checksum, _ wvm.Env, _ []byte, store wasmvm.KVStore, _ wasmvm.GoAPI, _ wasmvm.Querier, _ wasmvm.GasMeter, _ uint64, _ wvm.UFraction) (*wvm.Response, uint64, error) {
					return nil, 0, wasmtesting.ErrMockContract
				})
			},
			sdkioerrors.Wrap(types.ErrWasmContractCallFailed, wasmtesting.ErrMockContract.Error()),
			nil,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// reset suite to create fresh application state
			suite.SetupWasmWithMockVM()
			expectedClientStateBz = nil

			endpoint := wasmtesting.NewWasmEndpoint(suite.chainA)
			err := endpoint.CreateClient()
			suite.Require().NoError(err)

			store := suite.chainA.App.GetIBCKeeper().ClientKeeper.ClientStore(suite.chainA.GetContext(), endpoint.ClientID)
			clientMsg = &types.ClientMessage{
				Data: clienttypes.MustMarshalClientMessage(suite.chainA.App.AppCodec(), wasmtesting.MockTendermintClientMisbehaviour),
			}
			clientState := endpoint.GetClientState()

			tc.malleate()

			if tc.panicErr == nil {
				clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), store, clientMsg)
				if expectedClientStateBz != nil {
					suite.Require().Equal(expectedClientStateBz, store.Get(host.ClientStateKey()))
				}
			} else {
				suite.Require().PanicsWithError(tc.panicErr.Error(), func() {
					clientState.UpdateStateOnMisbehaviour(suite.chainA.GetContext(), suite.chainA.App.AppCodec(), store, clientMsg)
				})
			}
		})
	}
}

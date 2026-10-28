package xconsensus_test

import (
	"encoding/hex"
	"encoding/json"
	"math/rand"
	"slices"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	tmdb "github.com/cometbft/cometbft-db"
	"github.com/stretchr/testify/require"

	abci "github.com/NibiruChain/nibiru/v2/bft/abci/types"
	"github.com/NibiruChain/nibiru/v2/bft/libs/log"
	cmtproto "github.com/NibiruChain/nibiru/v2/bft/proto/tendermint/types"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/ed25519"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/store/prefix"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/sims"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
	authtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/auth/types"
	banktypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/bank/types"
	genutiltypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/genutil/types"
	govtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/gov/types"
	stakingtypes "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/x/staking/types"
	consensustypes "github.com/NibiruChain/nibiru/v2/x/consensus/types"

	"github.com/NibiruChain/nibiru/v2/app"
	"github.com/NibiruChain/nibiru/v2/app/appconst"
	"github.com/NibiruChain/nibiru/v2/eth"
	"github.com/NibiruChain/nibiru/v2/x/epochs"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testapp"
	"github.com/NibiruChain/nibiru/v2/x/nutil/testutil"
	"github.com/NibiruChain/nibiru/v2/x/sudo"
	wasmkeeper "github.com/NibiruChain/nibiru/v2/x/wasm/keeper"
	wasmtypes "github.com/NibiruChain/nibiru/v2/x/wasm/types"
	"github.com/NibiruChain/nibiru/v2/x/xconsensus"
)

// chain drives a test app block by block through the real ABCI methods, so
// the x/xconsensus EndBlocker, baseapp's EndBlock response and Commit all run
// as they do on a node.
type chain struct {
	t   *testing.T
	app *app.NibiruApp
	ctx sdk.Context
}

// newChain returns a fresh test chain at height 1, after InitChain.
func newChain(t *testing.T) *chain {
	t.Helper()
	nibiru, _ := testapp.NewNibiruTestApp(app.GenesisState{})
	return &chain{t: t, app: nibiru, ctx: testapp.NewContext(nibiru)}
}

// newChainWithDB is newChain on a caller-owned database (restart tests).
func newChainWithDB(t *testing.T, db tmdb.DB) *chain {
	t.Helper()
	nibiru, _ := testapp.NewNibiruTestAppWithDB(db, app.GenesisState{})
	return &chain{t: t, app: nibiru, ctx: testapp.NewContext(nibiru)}
}

// nextBlock ends and commits the current block, begins the next one, and
// returns the EndBlock response of the block it ended.
func (c *chain) nextBlock() abci.ResponseEndBlock {
	c.t.Helper()
	res := c.app.EndBlock(abci.RequestEndBlock{Height: c.ctx.BlockHeight()})
	c.app.Commit()
	header := cmtproto.Header{
		Height: c.ctx.BlockHeight() + 1,
		Time:   c.ctx.BlockTime().Add(time.Second),
	}
	c.app.BeginBlock(abci.RequestBeginBlock{Header: header})
	c.ctx = c.app.NewContext(false, header)
	return res
}

// contract returns the x-consensus address and fails if it is missing.
func (c *chain) contract() sdk.AccAddress {
	c.t.Helper()
	addr, found := c.app.XConsensusKeeper.ContractAddr(c.ctx)
	require.True(c.t, found, "x-consensus not found")
	return addr
}

// contractsByCreator lists every contract created by xconsensus.CreatorAddr.
func (c *chain) contractsByCreator() []sdk.AccAddress {
	var addrs []sdk.AccAddress
	c.app.WasmKeeper.IterateContractsByCreator(c.ctx, xconsensus.CreatorAddr(),
		func(addr sdk.AccAddress) bool {
			addrs = append(addrs, addr)
			return false
		})
	return addrs
}

// contractParams queries the contract with an unbounded query, as a client
// would.
func (c *chain) contractParams() xconsensus.ConsensusParams {
	c.t.Helper()
	req, err := json.Marshal(xconsensus.QueryParams())
	require.NoError(c.t, err)
	resp, err := c.app.WasmKeeper.QuerySmart(c.ctx, c.contract(), req)
	require.NoError(c.t, err)
	var params xconsensus.ConsensusParams
	require.NoError(c.t, json.Unmarshal(resp, &params))
	return params
}

// storedParams returns the x/consensus store copy, which baseapp reads.
func (c *chain) storedParams() cmtproto.ConsensusParams {
	c.t.Helper()
	cp := c.app.GetConsensusParams(c.ctx)
	require.NotNil(c.t, cp)
	return *cp
}

// update executes UpdateParams as sender in the current block.
func (c *chain) update(sender sdk.AccAddress, params xconsensus.ConsensusParams) error {
	c.t.Helper()
	return c.execute(sender, xconsensus.ExecuteMsg{
		UpdateParams: &xconsensus.UpdateParamsMsg{Params: params},
	})
}

// execute sends msg to x-consensus as sender in the current block, the way
// a MsgExecuteContract tx would.
func (c *chain) execute(sender sdk.AccAddress, msg xconsensus.ExecuteMsg) error {
	c.t.Helper()
	bz, err := json.Marshal(msg)
	require.NoError(c.t, err)
	_, err = wasmkeeper.NewDefaultPermissionKeeper(&c.app.WasmKeeper).
		Execute(c.ctx, c.contract(), sender, bz, sdk.NewCoins())
	return err
}

// migrateMsg is ExecuteMsg.Migrate to codeID with an empty migrate message.
func migrateMsg(codeID uint64) xconsensus.ExecuteMsg {
	return xconsensus.ExecuteMsg{Migrate: &xconsensus.MigrateExecMsg{
		CodeID: codeID, Msg: []byte(`{}`),
	}}
}

// storeCode uploads a second copy of the x-consensus code as sender, with
// the given instantiate permission, and returns its code ID.
func (c *chain) storeCode(
	sender sdk.AccAddress, access wasmtypes.AccessConfig,
) uint64 {
	c.t.Helper()
	codeID, _, err := wasmkeeper.NewDefaultPermissionKeeper(&c.app.WasmKeeper).
		Create(c.ctx, sender, xconsensus.XConsensusWasm, &access)
	require.NoError(c.t, err)
	return codeID
}

// isDeployer reports whether addr is listed in the x/sudo wasm_deployer
// role. It reads the role list directly, so the root does not count.
func (c *chain) isDeployer(addr sdk.AccAddress) bool {
	c.t.Helper()
	sudoers, err := c.app.SudoKeeper.Sudoers.Get(c.ctx)
	require.NoError(c.t, err)
	for _, role := range sudoers.Roles {
		if role.Role == sudo.RoleWasmDeployer {
			return slices.Contains(role.Members, addr.String())
		}
	}
	return false
}

// contractInfo returns the Wasm contract info of x-consensus.
func (c *chain) contractInfo() *wasmtypes.ContractInfo {
	c.t.Helper()
	info := c.app.WasmKeeper.GetContractInfo(c.ctx, c.contract())
	require.NotNil(c.t, info)
	return info
}

// writeContractParams overwrites the contract's stored params with raw JSON,
// bypassing its validation. It models a buggy contract or a bad migration.
func (c *chain) writeContractParams(raw string) {
	c.t.Helper()
	store := prefix.NewStore(
		c.ctx.KVStore(c.app.GetKey(wasmtypes.StoreKey)),
		wasmtypes.GetContractStorePrefix(c.contract()))
	store.Set([]byte("params"), []byte(raw))
}

// rootAddr is the x/sudo root of every test app.
func rootAddr() sdk.AccAddress { return sdk.MustAccAddressFromBech32(testutil.ADDR_SUDO_ROOT) }

// genesisParams is the x/consensus genesis of every test app, as contract
// params.
func genesisParams(t *testing.T) xconsensus.ConsensusParams {
	t.Helper()
	params, err := xconsensus.ConsensusParamsFromProto(*sims.DefaultConsensusParams)
	require.NoError(t, err)
	return params
}

// changedParams returns genesis params with a new block gas limit, an extra
// key type, and the commit wait set to waitMs.
func changedParams(t *testing.T, waitMs uint64) xconsensus.ConsensusParams {
	t.Helper()
	params := genesisParams(t)
	params.Block.MaxGas = 77_000_000
	params.Validator.PubKeyTypes = []string{"ed25519", "secp256k1"}
	params.MinBlockCommitDurationMs = &waitMs
	return params
}

// mustProto converts contract params to the engine type with version.
func mustProto(
	t *testing.T, p xconsensus.ConsensusParams, version *cmtproto.VersionParams,
) cmtproto.ConsensusParams {
	t.Helper()
	pb, err := p.ValidatedProto(version)
	require.NoError(t, err)
	return pb
}

// TestGenesisCreatesXConsensus checks a fresh chain: InitChain creates exactly
// one x-consensus, created by the module address, as its own Wasm admin and a
// member of the x/sudo wasm_deployer role, code instantiable by nobody else,
// the genesis params, and the commit wait unset. testapp's genesis bonds its validator through staking genesis;
// gentxs are covered by the localnet check, which uses real gentxs.
func TestGenesisCreatesXConsensus(t *testing.T) {
	c := newChain(t)
	require.Len(t, c.contractsByCreator(), 1)

	info := c.app.WasmKeeper.GetContractInfo(c.ctx, c.contract())
	require.NotNil(t, info)
	require.Equal(t, xconsensus.CreatorAddr().String(), info.Creator)
	require.Equal(t, c.contract().String(), info.Admin, "x-consensus must be its own admin")
	require.True(t, c.isDeployer(c.contract()), "x-consensus must hold wasm_deployer")
	require.Equal(t, xconsensus.ContractLabel, info.Label)
	code := c.app.WasmKeeper.GetCodeInfo(c.ctx, info.CodeID)
	require.Equal(t, wasmtypes.AccessTypeNobody, code.InstantiateConfig.Permission)
	require.Equal(t, xconsensus.XConsensusWasmSHA256, hex.EncodeToString(code.CodeHash))

	require.Equal(t, genesisParams(t), c.contractParams())
	require.Equal(t, *sims.DefaultConsensusParams, c.storedParams())
	_, ok := c.app.XConsensusKeeper.CommitWait()
	require.False(t, ok)
	_, ok = c.app.CommitTimeoutFunc()(2)
	require.False(t, ok)

	// Blocks with an unchanged contract leave the store unchanged.
	res := c.nextBlock()
	require.Equal(t, *sims.DefaultConsensusParams, *res.ConsensusParamUpdates)
	require.Equal(t, *sims.DefaultConsensusParams, c.storedParams())

	// EnsureXConsensus is idempotent.
	require.NoError(t, c.app.XConsensusKeeper.EnsureXConsensus(c.ctx))
	require.Len(t, c.contractsByCreator(), 1)
}

// TestPredictedAddress checks that PredictContractAddr, computed from the
// embedded checksum alone, is the address genesis created. Operators and docs
// can publish this address before the chain or upgrade creates it.
func TestPredictedAddress(t *testing.T) {
	c := newChain(t)
	checksum, err := hex.DecodeString(xconsensus.XConsensusWasmSHA256)
	require.NoError(t, err)
	predicted := xconsensus.PredictContractAddr(checksum)
	require.Equal(t, predicted, c.contract())
	require.Equal(t, predicted.String(), c.contractInfo().Admin)
	t.Logf("x-consensus address: %s (creator %s)",
		predicted, xconsensus.CreatorAddr())
}

// TestSelfAdminMigrate checks the self-admin model: the x/sudo root migrates
// x-consensus by executing Migrate on the contract, which then migrates
// itself. The code ID changes and the params survive. The new code must let
// the contract instantiate it, because the Wasm keeper checks the caller,
// here the contract, against the new code's instantiate permission.
func TestSelfAdminMigrate(t *testing.T) {
	c := newChain(t)
	want := changedParams(t, 250)
	require.NoError(t, c.update(rootAddr(), want))
	oldCodeID := c.contractInfo().CodeID

	closed := c.storeCode(rootAddr(), wasmtypes.AllowNobody)
	require.ErrorContains(t, c.execute(rootAddr(), migrateMsg(closed)),
		"to use new code")

	newCodeID := c.storeCode(rootAddr(),
		wasmtypes.AccessTypeAnyOfAddresses.With(c.contract()))
	require.NotEqual(t, oldCodeID, newCodeID)
	require.NoError(t, c.execute(rootAddr(), migrateMsg(newCodeID)))

	info := c.contractInfo()
	require.Equal(t, newCodeID, info.CodeID)
	require.Equal(t, c.contract().String(), info.Admin)
	require.Equal(t, want, c.contractParams())

	// The migrated contract still serves the EndBlocker.
	res := c.nextBlock()
	require.Equal(t, mustProto(t, want, nil), *res.ConsensusParamUpdates)
}

// TestSelfAdminRootOnly checks that only the live x/sudo root can migrate
// x-consensus or move its admin, and that a root rotation moves that right
// at once, with no MsgUpdateAdmin.
func TestSelfAdminRootOnly(t *testing.T) {
	c := newChain(t)
	codeID := c.storeCode(rootAddr(),
		wasmtypes.AccessTypeAnyOfAddresses.With(c.contract()))
	outsider := testutil.NewAccAddress()
	oldCodeID := c.contractInfo().CodeID

	require.ErrorContains(t, c.execute(outsider, migrateMsg(codeID)),
		"is not the x/sudo root")
	require.ErrorContains(t, c.execute(outsider, xconsensus.ExecuteMsg{
		UpdateAdmin: &xconsensus.UpdateAdminMsg{Admin: outsider.String()},
	}), "is not the x/sudo root")
	// Sending MsgMigrateContract directly fails too: the admin is the
	// contract, not the root.
	_, err := wasmkeeper.NewDefaultPermissionKeeper(&c.app.WasmKeeper).
		Migrate(c.ctx, c.contract(), rootAddr(), codeID, []byte(`{}`))
	require.ErrorContains(t, err, "can not migrate")
	require.Equal(t, oldCodeID, c.contractInfo().CodeID)
	require.Equal(t, c.contract().String(), c.contractInfo().Admin)

	// After a root rotation the old root is out and the new root is in.
	newRoot := testutil.NewAccAddress()
	sudoers, err := c.app.SudoKeeper.Sudoers.Get(c.ctx)
	require.NoError(t, err)
	sudoers.Root = newRoot.String()
	c.app.SudoKeeper.Sudoers.Set(c.ctx, sudoers)
	require.ErrorContains(t, c.execute(rootAddr(), migrateMsg(codeID)),
		"is not the x/sudo root")
	require.NoError(t, c.execute(newRoot, migrateMsg(codeID)))
	require.Equal(t, codeID, c.contractInfo().CodeID)
}

// TestSelfAdminUpdateAdmin checks that the root can hand the Wasm admin to
// another address, after which Migrate through the contract stops working
// and the new admin migrates directly.
func TestSelfAdminUpdateAdmin(t *testing.T) {
	c := newChain(t)
	newAdmin := testutil.NewAccAddress()
	require.NoError(t, c.execute(rootAddr(), xconsensus.ExecuteMsg{
		UpdateAdmin: &xconsensus.UpdateAdminMsg{Admin: newAdmin.String()},
	}))
	require.Equal(t, newAdmin.String(), c.contractInfo().Admin)

	codeID := c.storeCode(rootAddr(), wasmtypes.AllowEverybody)
	require.ErrorContains(t, c.execute(rootAddr(), migrateMsg(codeID)),
		"can not migrate")
	_, err := wasmkeeper.NewDefaultPermissionKeeper(&c.app.WasmKeeper).
		Migrate(c.ctx, c.contract(), newAdmin, codeID, []byte(`{}`))
	require.NoError(t, err)
	require.Equal(t, codeID, c.contractInfo().CodeID)
}

// TestSelfAdminMigrateMainnetGuard checks Migrate under the mainnet Wasm
// deployer guard. The guard checks the actor of the migration, which is the
// contract, not the root, so EnsureXConsensus grants the contract the x/sudo
// wasm_deployer role. Migrate then works on cataclysm-1 with no manual step.
func TestSelfAdminMigrateMainnetGuard(t *testing.T) {
	c := newChain(t)
	c.ctx = c.ctx.WithChainID(appconst.SDK_CHAIN_ID_MAINNET)
	require.True(t, c.isDeployer(c.contract()))
	codeID := c.storeCode(rootAddr(),
		wasmtypes.AccessTypeAnyOfAddresses.With(c.contract()))
	require.NoError(t, c.execute(rootAddr(), migrateMsg(codeID)))
	require.Equal(t, codeID, c.contractInfo().CodeID)
}

// TestEnsureXConsensusRestoresDeployerRole models a chain whose contract
// exists without the wasm_deployer role, as one created before the grant
// existed would be: mainnet Migrate fails, and the next EnsureXConsensus
// grants the role without creating a second contract. Repeating it changes
// nothing.
func TestEnsureXConsensusRestoresDeployerRole(t *testing.T) {
	c := newChain(t)
	c.ctx = c.ctx.WithChainID(appconst.SDK_CHAIN_ID_MAINNET)
	_, err := c.app.SudoKeeper.UpdateRoleMembers(sdk.WrapSDKContext(c.ctx),
		&sudo.MsgUpdateRoleMembers{
			Sender: testutil.ADDR_SUDO_ROOT, Role: sudo.RoleWasmDeployer,
			Remove: []string{c.contract().String()},
		})
	require.NoError(t, err)
	require.False(t, c.isDeployer(c.contract()))

	codeID := c.storeCode(rootAddr(),
		wasmtypes.AccessTypeAnyOfAddresses.With(c.contract()))
	require.ErrorContains(t, c.execute(rootAddr(), migrateMsg(codeID)),
		"requires root or role")

	require.NoError(t, c.app.XConsensusKeeper.EnsureXConsensus(c.ctx))
	require.True(t, c.isDeployer(c.contract()))
	before, err := c.app.SudoKeeper.Sudoers.Get(c.ctx)
	require.NoError(t, err)
	require.NoError(t, c.app.XConsensusKeeper.EnsureXConsensus(c.ctx))
	after, err := c.app.SudoKeeper.Sudoers.Get(c.ctx)
	require.NoError(t, err)
	require.Equal(t, before, after)
	require.Len(t, c.contractsByCreator(), 1)

	require.NoError(t, c.execute(rootAddr(), migrateMsg(codeID)))
	require.Equal(t, codeID, c.contractInfo().CodeID)
}

// TestParamChangeAppliesFromNextBlock follows one UpdateParams executed by the
// x/sudo root in block H: the store is unchanged during H, EndBlock H returns
// the new params to Nibiru BFT, the commit hook reports the new wait after H,
// and every reader in H+1 sees the new params. A non-root sender is rejected.
func TestParamChangeAppliesFromNextBlock(t *testing.T) {
	c := newChain(t)
	h := c.ctx.BlockHeight()
	want := changedParams(t, 200)

	require.ErrorContains(t, c.update(testutil.NewAccAddress(), want),
		"is not the x/sudo root")
	require.NoError(t, c.update(rootAddr(), want))

	// During H, readers still see the old params.
	require.Equal(t, *sims.DefaultConsensusParams, c.storedParams())
	_, ok := c.app.XConsensusKeeper.CommitWait()
	require.False(t, ok)

	res := c.nextBlock()
	require.Equal(t, h+1, c.ctx.BlockHeight())
	wantProto := mustProto(t, want, nil)
	require.Equal(t, wantProto, *res.ConsensusParamUpdates,
		"EndBlock H must return the new params")
	require.Equal(t, wantProto, c.storedParams(), "H+1 must read the new params")
	require.Equal(t, uint64(77_000_000), c.app.GetMaximumBlockGas(c.ctx))

	wait, ok := c.app.CommitTimeoutFunc()(h + 1)
	require.True(t, ok)
	require.Equal(t, 200*time.Millisecond, wait)

	// Unsetting the wait hands pacing back to config.toml.
	want.MinBlockCommitDurationMs = nil
	require.NoError(t, c.update(rootAddr(), want))
	c.nextBlock()
	_, ok = c.app.CommitTimeoutFunc()(h + 2)
	require.False(t, ok)
}

// TestRefreshKeepsVersion checks that the refresh copies Version from the
// store. The contract has no version, and the engine writes Version.App into
// block headers, so a refresh must never zero it.
func TestRefreshKeepsVersion(t *testing.T) {
	c := newChain(t)
	withVersion := *sims.DefaultConsensusParams
	withVersion.Version = &cmtproto.VersionParams{App: 7}
	c.app.ConsensusParamsKeeper.Set(c.ctx, &withVersion)

	require.NoError(t, c.update(rootAddr(), changedParams(t, 500)))
	res := c.nextBlock()
	require.Equal(t, uint64(7), res.ConsensusParamUpdates.Version.App)
	require.Equal(t, uint64(7), c.storedParams().Version.App)
	require.Equal(t, int64(77_000_000), c.storedParams().Block.MaxGas)
}

// TestRefreshFallback breaks the contract in three ways after a good update
// and checks that each time the chain keeps producing blocks with the last
// good params and commit wait, without panicking.
func TestRefreshFallback(t *testing.T) {
	cases := []struct {
		name    string
		breakIt func(c *chain)
	}{
		{
			name: "query error",
			breakIt: func(c *chain) {
				c.writeContractParams(`{"not":"params"}`)
			},
		},
		{
			name: "out of gas",
			breakIt: func(c *chain) {
				// The real query uses far more than 1000 gas, so this behaves
				// like a contract that loops.
				c.app.XConsensusKeeper.SetQueryGasLimit(1_000)
			},
		},
		{
			// The contract validates like the engine, so an invalid value can
			// only appear through a bug or a bad migration.
			name: "invalid values",
			breakIt: func(c *chain) {
				params := changedParams(t, 50)
				params.Validator.PubKeyTypes = []string{"bogus"}
				bz, err := json.Marshal(params)
				require.NoError(t, err)
				c.writeContractParams(string(bz))
			},
		},
		{
			name: "commit wait above the cap",
			breakIt: func(c *chain) {
				params := changedParams(t, xconsensus.MaxMinBlockCommitDurationMs+1)
				bz, err := json.Marshal(params)
				require.NoError(t, err)
				c.writeContractParams(string(bz))
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := newChain(t)
			good := changedParams(t, 300)
			require.NoError(t, c.update(rootAddr(), good))
			c.nextBlock()
			goodProto := mustProto(t, good, nil)
			require.Equal(t, goodProto, c.storedParams())

			tc.breakIt(c)
			for i := 0; i < 3; i++ {
				var res abci.ResponseEndBlock
				require.NotPanics(t, func() { res = c.nextBlock() })
				require.Equal(t, goodProto, *res.ConsensusParamUpdates)
				require.Equal(t, goodProto, c.storedParams())
				wait, ok := c.app.XConsensusKeeper.CommitWait()
				require.True(t, ok)
				require.Equal(t, 300*time.Millisecond, wait)
			}
			require.Error(t, c.app.XConsensusKeeper.Refresh(c.ctx))
		})
	}
}

// TestValidatedProto checks the Go validation path directly: it applies
// Nibiru BFT's ValidateBasic to the full struct, the commit wait cap, and
// keeps the caller's version.
func TestValidatedProto(t *testing.T) {
	good := changedParams(t, xconsensus.MaxMinBlockCommitDurationMs)
	pb, err := good.ValidatedProto(&cmtproto.VersionParams{App: 3})
	require.NoError(t, err)
	require.Equal(t, uint64(3), pb.Version.App)
	pb, err = good.ValidatedProto(nil)
	require.NoError(t, err)
	require.Nil(t, pb.Version)

	for name, mutate := range map[string]func(p *xconsensus.ConsensusParams){
		"commit wait above cap": func(p *xconsensus.ConsensusParams) {
			ms := xconsensus.MaxMinBlockCommitDurationMs + 1
			p.MinBlockCommitDurationMs = &ms
		},
		"zero max bytes":      func(p *xconsensus.ConsensusParams) { p.Block.MaxBytes = 0 },
		"max gas below -1":    func(p *xconsensus.ConsensusParams) { p.Block.MaxGas = -2 },
		"zero evidence age":   func(p *xconsensus.ConsensusParams) { p.Evidence.MaxAgeNumBlocks = 0 },
		"evidence over block": func(p *xconsensus.ConsensusParams) { p.Evidence.MaxBytes = p.Block.MaxBytes + 1 },
		"no key types":        func(p *xconsensus.ConsensusParams) { p.Validator.PubKeyTypes = nil },
		"unknown key type":    func(p *xconsensus.ConsensusParams) { p.Validator.PubKeyTypes = []string{"sr25519"} },
	} {
		t.Run(name, func(t *testing.T) {
			p := changedParams(t, 0)
			mutate(&p)
			_, err := p.ValidatedProto(nil)
			require.Error(t, err)
		})
	}
}

// TestRestartKeepsParamsAndWarmsCommitWait opens a second app on the same
// database after a param change was committed, as a node restart or state
// sync would. The new instance has run no EndBlock, yet its first block reads
// the new params from the store, and the commit hook already reports the
// contract's wait (warmed in NewNibiruApp), so the first height after the
// restart does not fall back to config.toml.
func TestRestartKeepsParamsAndWarmsCommitWait(t *testing.T) {
	db := tmdb.NewMemDB()
	c := newChainWithDB(t, db)
	want := changedParams(t, 300)
	require.NoError(t, c.update(rootAddr(), want))
	c.nextBlock()
	lastHeight := c.app.LastBlockHeight()

	restarted := app.NewNibiruApp(
		log.NewNopLogger(), db, nil, true, sims.EmptyAppOptions{})
	require.Equal(t, lastHeight, restarted.LastBlockHeight())

	wait, ok := restarted.CommitTimeoutFunc()(lastHeight + 1)
	require.True(t, ok)
	require.Equal(t, 300*time.Millisecond, wait)

	header := cmtproto.Header{Height: lastHeight + 1, Time: time.Now().UTC()}
	restarted.BeginBlock(abci.RequestBeginBlock{Header: header})
	ctx := restarted.NewContext(false, header)
	require.Equal(t, mustProto(t, want, nil), *restarted.GetConsensusParams(ctx))
	require.Equal(t, uint64(77_000_000), restarted.GetMaximumBlockGas(ctx))

	res := restarted.EndBlock(abci.RequestEndBlock{Height: header.Height})
	require.Equal(t, mustProto(t, want, nil), *res.ConsensusParamUpdates)
}

// TestExportImportRoundTrip exports a chain whose params were changed and
// imports it into a new app. The imported Wasm genesis rebuilds the creator
// index, so InitChain finds the existing contract instead of creating a
// second one, the params match, and the commit wait is warmed at genesis.
func TestExportImportRoundTrip(t *testing.T) {
	c := newChain(t)
	want := changedParams(t, 400)
	require.NoError(t, c.update(rootAddr(), want))
	c.nextBlock()
	addr := c.contract()

	exported, err := c.app.ExportAppStateAndValidators(false, nil, nil)
	require.NoError(t, err)
	require.Equal(t, mustProto(t, want, nil), *exported.ConsensusParams)

	imported := app.NewNibiruApp(
		log.NewNopLogger(), tmdb.NewMemDB(), nil, true, sims.EmptyAppOptions{})
	imported.InitChain(abci.RequestInitChain{
		Time:            time.Now().UTC(),
		ConsensusParams: exported.ConsensusParams,
		AppStateBytes:   exported.AppState,
	})
	c2 := &chain{t: t, app: imported, ctx: testapp.NewContext(imported)}

	require.Equal(t, []sdk.AccAddress{addr}, c2.contractsByCreator())
	require.Equal(t, addr, c2.contract())
	require.Equal(t, want, c2.contractParams())
	require.Equal(t, mustProto(t, want, nil), c2.storedParams())
	wait, ok := imported.CommitTimeoutFunc()(1)
	require.True(t, ok)
	require.Equal(t, 400*time.Millisecond, wait)

	res := c2.nextBlock()
	require.Equal(t, mustProto(t, want, nil), *res.ConsensusParamUpdates)
}

// TestConsensusMsgUpdateParamsBlocked checks the defense in depth around
// x/consensus: its MsgUpdateParams fails in the router even with the gov
// authority, which is how gov proposals execute it, and a write that reaches
// the store anyway is overwritten by the refresh in the same block.
func TestConsensusMsgUpdateParamsBlocked(t *testing.T) {
	c := newChain(t)
	tampered := *sims.DefaultConsensusParams
	tampered.Block = &cmtproto.BlockParams{MaxBytes: 1_000, MaxGas: 1}

	msg := &consensustypes.MsgUpdateParams{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Block:     tampered.Block,
		Evidence:  &cmtproto.EvidenceParams{MaxAgeNumBlocks: 1, MaxAgeDuration: time.Hour, MaxBytes: 0},
		Validator: tampered.Validator,
	}
	handler := c.app.MsgServiceRouter().Handler(msg)
	require.NotNil(t, handler, "the route stays registered; the circuit rejects it")
	_, err := handler(c.ctx, msg)
	require.ErrorContains(t, err, "circuit breaker")
	require.Equal(t, *sims.DefaultConsensusParams, c.storedParams())

	// A direct store write in the block is overwritten at its EndBlock.
	c.app.ConsensusParamsKeeper.Set(c.ctx, &tampered)
	res := c.nextBlock()
	require.Equal(t, *sims.DefaultConsensusParams, *res.ConsensusParamUpdates)
	require.Equal(t, *sims.DefaultConsensusParams, c.storedParams())

	// The x/consensus Params query is still routed.
	route := c.app.GRPCQueryRouter().Route("/cosmos.consensus.v1.Query/Params")
	require.NotNil(t, route)
	reqBz, err := (&consensustypes.QueryParamsRequest{}).Marshal()
	require.NoError(t, err)
	queryRes, err := route(c.ctx, abci.RequestQuery{Data: reqBz})
	require.NoError(t, err)
	var resp consensustypes.QueryParamsResponse
	require.NoError(t, resp.Unmarshal(queryRes.Value))
	require.Equal(t, int64(100_000_000), resp.Params.Block.MaxGas)
}

// TestQueryGasUsage measures the per-block params query, which justifies
// QueryGasLimit, and checks the limit keeps a wide margin over it.
func TestQueryGasUsage(t *testing.T) {
	c := newChain(t)
	req, err := json.Marshal(xconsensus.QueryParams())
	require.NoError(t, err)

	start := time.Now()
	_, gasUsed, err := c.app.WasmKeeper.QuerySmartBounded(
		c.ctx, c.contract(), req, xconsensus.QueryGasLimit)
	elapsed := time.Since(start)
	require.NoError(t, err)
	t.Logf("x-consensus params query: %d gas, %s", gasUsed, elapsed)
	require.Less(t, gasUsed*10, xconsensus.QueryGasLimit,
		"QueryGasLimit should keep a 10x margin over the measured query")
}

// TestGenesisCost reports how much creating x-consensus adds to InitChain,
// which every test app pays. It stores and instantiates a second copy under
// another creator, which is the same work EnsureXConsensus does, and times a
// whole test app for comparison.
func TestGenesisCost(t *testing.T) {
	start := time.Now()
	c := newChain(t)
	appTotal := time.Since(start)

	ctx, _ := c.ctx.CacheContext()
	gov := wasmkeeper.NewGovPermissionKeeper(&c.app.WasmKeeper)
	initMsg, err := json.Marshal(xconsensus.InstantiateMsg{Params: genesisParams(t)})
	require.NoError(t, err)
	creator := testutil.NewAccAddress()

	start = time.Now()
	codeID, _, err := gov.Create(ctx, creator, xconsensus.XConsensusWasm, nil)
	require.NoError(t, err)
	_, _, err = gov.Instantiate2(ctx, codeID, creator, nil, initMsg,
		"cost", sdk.NewCoins(), []byte("cost"), false)
	require.NoError(t, err)
	ensure := time.Since(start)
	t.Logf("test app with InitChain: %s; x-consensus store+instantiate: %s",
		appTotal, ensure)
}

// TestGenesisWithGentx boots a chain the way `nibid genesis gentx` and
// `collect-gentxs` do: no validators in staking genesis, one signed
// MsgCreateValidator in genutil. The gentx runs during module genesis and
// reads the allowed key types from the x/consensus store, which baseapp fills
// before any module genesis; x-consensus is created afterwards, seeded from
// the same store.
func TestGenesisWithGentx(t *testing.T) {
	nibiru := app.NewNibiruApp(
		log.NewNopLogger(), tmdb.NewMemDB(), nil, true, sims.EmptyAppOptions{})
	encoding := app.MakeEncodingConfig()
	cdc := nibiru.AppCodec()
	gen := nibiru.DefaultGenesis()

	gen[epochs.ModuleName] = cdc.MustMarshalJSON(
		epochs.DefaultGenesisFromTime(time.Now().UTC()))
	gen[sudo.ModuleName] = cdc.MustMarshalJSON(&sudo.GenesisState{
		Sudoers: sudo.Sudoers{Root: testutil.ADDR_SUDO_ROOT},
	})

	priv := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(priv.PubKey().Address())
	acc := authtypes.NewBaseAccount(addr, priv.PubKey(), 0, 0)
	gen[authtypes.ModuleName] = cdc.MustMarshalJSON(authtypes.NewGenesisState(
		authtypes.DefaultParams(), []authtypes.GenesisAccount{acc}))
	bond := sdk.NewCoin(appconst.DENOM_UNIBI, sdk.DefaultPowerReduction.MulRaw(10))
	var bankGen banktypes.GenesisState
	cdc.MustUnmarshalJSON(gen[banktypes.ModuleName], &bankGen)
	bankGen.Balances = append(bankGen.Balances, banktypes.Balance{
		Address: addr.String(), Coins: sdk.NewCoins(bond),
	})
	gen[banktypes.ModuleName] = cdc.MustMarshalJSON(&bankGen)

	msg, err := stakingtypes.NewMsgCreateValidator(
		sdk.ValAddress(addr), ed25519.GenPrivKey().PubKey(), bond,
		stakingtypes.Description{Moniker: "gentx"},
		stakingtypes.NewCommissionRates(
			sdkmath.LegacyNewDecWithPrec(1, 1), sdkmath.LegacyNewDecWithPrec(2, 1),
			sdkmath.LegacyNewDecWithPrec(1, 2)),
		sdkmath.OneInt())
	require.NoError(t, err)
	tx, err := sims.GenSignedMockTx(rand.New(rand.NewSource(1)),
		encoding.TxConfig, []sdk.Msg{msg}, sdk.NewCoins(), sims.DefaultGenTxGas,
		// The ante handler signs empty chain IDs as the EVM testnet ID.
		eth.EIP155ChainID_Testnet, []uint64{0}, []uint64{0}, priv)
	require.NoError(t, err)
	gen[genutiltypes.ModuleName] = cdc.MustMarshalJSON(
		genutiltypes.NewGenesisStateFromTx(encoding.TxConfig.TxJSONEncoder(), []sdk.Tx{tx}))

	stateBytes, err := json.Marshal(gen)
	require.NoError(t, err)
	res := nibiru.InitChain(abci.RequestInitChain{
		Time:            time.Now().UTC(),
		ConsensusParams: sims.DefaultConsensusParams,
		AppStateBytes:   stateBytes,
	})
	require.Len(t, res.Validators, 1, "the gentx validator must be bonded")

	c := &chain{t: t, app: nibiru, ctx: nibiru.NewContext(false, cmtproto.Header{
		Height: 1, Time: time.Now().UTC(),
	})}
	require.Len(t, c.contractsByCreator(), 1)
	require.Equal(t, genesisParams(t), c.contractParams())
	c.nextBlock()
}

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/internal/api/testdb"
	"github.com/NibiruChain/nibiru/v2/lib/wasmvm/wvm"
)

/** helper constructors **/

const MOCK_CONTRACT_ADDR = "contract"

func MockEnv() wvm.Env {
	return wvm.Env{
		Block: wvm.BlockInfo{
			Height:  123,
			Time:    1578939743_987654321,
			ChainID: "foobar",
		},
		Transaction: &wvm.TransactionInfo{
			Index: 4,
		},
		Contract: wvm.ContractInfo{
			Address: MOCK_CONTRACT_ADDR,
		},
	}
}

func MockEnvBin(t *testing.T) []byte {
	bin, err := json.Marshal(MockEnv())
	require.NoError(t, err)
	return bin
}

func MockInfo(sender wvm.HumanAddress, funds []wvm.Coin) wvm.MessageInfo {
	return wvm.MessageInfo{
		Sender: sender,
		Funds:  funds,
	}
}

func MockInfoWithFunds(sender wvm.HumanAddress) wvm.MessageInfo {
	return MockInfo(sender, []wvm.Coin{{
		Denom:  "ATOM",
		Amount: "100",
	}})
}

func MockInfoBin(t *testing.T, sender wvm.HumanAddress) []byte {
	bin, err := json.Marshal(MockInfoWithFunds(sender))
	require.NoError(t, err)
	return bin
}

func MockIBCChannel(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannel {
	return wvm.IBCChannel{
		Endpoint: wvm.IBCEndpoint{
			PortID:    "my_port",
			ChannelID: channelID,
		},
		CounterpartyEndpoint: wvm.IBCEndpoint{
			PortID:    "their_port",
			ChannelID: "channel-7",
		},
		Order:        ordering,
		Version:      ibcVersion,
		ConnectionID: "connection-3",
	}
}

func MockIBCChannelOpenInit(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelOpenMsg {
	return wvm.IBCChannelOpenMsg{
		OpenInit: &wvm.IBCOpenInit{
			Channel: MockIBCChannel(channelID, ordering, ibcVersion),
		},
		OpenTry: nil,
	}
}

func MockIBCChannelOpenTry(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelOpenMsg {
	return wvm.IBCChannelOpenMsg{
		OpenInit: nil,
		OpenTry: &wvm.IBCOpenTry{
			Channel:             MockIBCChannel(channelID, ordering, ibcVersion),
			CounterpartyVersion: ibcVersion,
		},
	}
}

func MockIBCChannelConnectAck(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelConnectMsg {
	return wvm.IBCChannelConnectMsg{
		OpenAck: &wvm.IBCOpenAck{
			Channel:             MockIBCChannel(channelID, ordering, ibcVersion),
			CounterpartyVersion: ibcVersion,
		},
		OpenConfirm: nil,
	}
}

func MockIBCChannelConnectConfirm(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelConnectMsg {
	return wvm.IBCChannelConnectMsg{
		OpenAck: nil,
		OpenConfirm: &wvm.IBCOpenConfirm{
			Channel: MockIBCChannel(channelID, ordering, ibcVersion),
		},
	}
}

func MockIBCChannelCloseInit(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelCloseMsg {
	return wvm.IBCChannelCloseMsg{
		CloseInit: &wvm.IBCCloseInit{
			Channel: MockIBCChannel(channelID, ordering, ibcVersion),
		},
		CloseConfirm: nil,
	}
}

func MockIBCChannelCloseConfirm(channelID string, ordering wvm.IBCOrder, ibcVersion string) wvm.IBCChannelCloseMsg {
	return wvm.IBCChannelCloseMsg{
		CloseInit: nil,
		CloseConfirm: &wvm.IBCCloseConfirm{
			Channel: MockIBCChannel(channelID, ordering, ibcVersion),
		},
	}
}

func MockIBCPacket(myChannel string, data []byte) wvm.IBCPacket {
	return wvm.IBCPacket{
		Data: data,
		Src: wvm.IBCEndpoint{
			PortID:    "their_port",
			ChannelID: "channel-7",
		},
		Dest: wvm.IBCEndpoint{
			PortID:    "my_port",
			ChannelID: myChannel,
		},
		Sequence: 15,
		Timeout: wvm.IBCTimeout{
			Block: &wvm.IBCTimeoutBlock{
				Revision: 1,
				Height:   123456,
			},
		},
	}
}

func MockIBCPacketReceive(myChannel string, data []byte) wvm.IBCPacketReceiveMsg {
	return wvm.IBCPacketReceiveMsg{
		Packet: MockIBCPacket(myChannel, data),
	}
}

func MockIBCPacketAck(myChannel string, data []byte, ack wvm.IBCAcknowledgement) wvm.IBCPacketAckMsg {
	packet := MockIBCPacket(myChannel, data)

	return wvm.IBCPacketAckMsg{
		Acknowledgement: ack,
		OriginalPacket:  packet,
	}
}

func MockIBCPacketTimeout(myChannel string, data []byte) wvm.IBCPacketTimeoutMsg {
	packet := MockIBCPacket(myChannel, data)

	return wvm.IBCPacketTimeoutMsg{
		Packet: packet,
	}
}

/*** Mock GasMeter ****/
// This code is borrowed from Cosmos-SDK store/types/gas.go

// ErrorOutOfGas defines an error thrown when an action results in out of gas.
type ErrorOutOfGas struct {
	Descriptor string
}

// ErrorGasOverflow defines an error thrown when an action results gas consumption
// unsigned integer overflow.
type ErrorGasOverflow struct {
	Descriptor string
}

type MockGasMeter interface {
	wvm.GasMeter
	ConsumeGas(amount wvm.Gas, descriptor string)
}

type mockGasMeter struct {
	limit    wvm.Gas
	consumed wvm.Gas
}

// NewMockGasMeter returns a reference to a new mockGasMeter.
func NewMockGasMeter(limit wvm.Gas) MockGasMeter {
	return &mockGasMeter{
		limit:    limit,
		consumed: 0,
	}
}

func (g *mockGasMeter) GasConsumed() wvm.Gas {
	return g.consumed
}

func (g *mockGasMeter) Limit() wvm.Gas {
	return g.limit
}

// addUint64Overflow performs the addition operation on two uint64 integers and
// returns a boolean on whether or not the result overflows.
func addUint64Overflow(a, b uint64) (uint64, bool) {
	if math.MaxUint64-a < b {
		return 0, true
	}

	return a + b, false
}

func (g *mockGasMeter) ConsumeGas(amount wvm.Gas, descriptor string) {
	var overflow bool
	// TODO: Should we set the consumed field after overflow checking?
	g.consumed, overflow = addUint64Overflow(g.consumed, amount)
	if overflow {
		panic(ErrorGasOverflow{descriptor})
	}

	if g.consumed > g.limit {
		panic(ErrorOutOfGas{descriptor})
	}
}

/*** Mock types.KVStore ****/
// Much of this code is borrowed from Cosmos-SDK store/transient.go

// Note: these gas prices are all in *wasmer gas* and (sdk gas * 100)
//
// We making simple values and non-clear multiples so it is easy to see their impact in test output
// Also note we do not charge for each read on an iterator (out of simplicity and not needed for tests)
const (
	GetPrice    uint64 = 99000
	SetPrice    uint64 = 187000
	RemovePrice uint64 = 142000
	RangePrice  uint64 = 261000
)

type Lookup struct {
	db    *testdb.MemDB
	meter MockGasMeter
}

func NewLookup(meter MockGasMeter) *Lookup {
	return &Lookup{
		db:    testdb.NewMemDB(),
		meter: meter,
	}
}

func (l *Lookup) SetGasMeter(meter MockGasMeter) {
	l.meter = meter
}

func (l *Lookup) WithGasMeter(meter MockGasMeter) *Lookup {
	return &Lookup{
		db:    l.db,
		meter: meter,
	}
}

// Get wraps the underlying DB's Get method panicing on error.
func (l Lookup) Get(key []byte) []byte {
	l.meter.ConsumeGas(GetPrice, "get")
	v, err := l.db.Get(key)
	if err != nil {
		panic(err)
	}

	return v
}

// Set wraps the underlying DB's Set method panicing on error.
func (l Lookup) Set(key, value []byte) {
	l.meter.ConsumeGas(SetPrice, "set")
	if err := l.db.Set(key, value); err != nil {
		panic(err)
	}
}

// Delete wraps the underlying DB's Delete method panicing on error.
func (l Lookup) Delete(key []byte) {
	l.meter.ConsumeGas(RemovePrice, "remove")
	if err := l.db.Delete(key); err != nil {
		panic(err)
	}
}

// Iterator wraps the underlying DB's Iterator method panicing on error.
func (l Lookup) Iterator(start, end []byte) wvm.Iterator {
	l.meter.ConsumeGas(RangePrice, "range")
	iter, err := l.db.Iterator(start, end)
	if err != nil {
		panic(err)
	}

	return iter
}

// ReverseIterator wraps the underlying DB's ReverseIterator method panicing on error.
func (l Lookup) ReverseIterator(start, end []byte) wvm.Iterator {
	l.meter.ConsumeGas(RangePrice, "range")
	iter, err := l.db.ReverseIterator(start, end)
	if err != nil {
		panic(err)
	}

	return iter
}

var _ wvm.KVStore = (*Lookup)(nil)

/***** Mock types.GoAPI ****/

const CanonicalLength = 32

const (
	CostCanonical uint64 = 440
	CostHuman     uint64 = 550
)

func MockCanonicalAddress(human string) ([]byte, uint64, error) {
	if len(human) > CanonicalLength {
		return nil, 0, fmt.Errorf("human encoding too long")
	}
	res := make([]byte, CanonicalLength)
	copy(res, []byte(human))
	return res, CostCanonical, nil
}

func MockHumanAddress(canon []byte) (string, uint64, error) {
	if len(canon) != CanonicalLength {
		return "", 0, fmt.Errorf("wrong canonical length")
	}
	cut := CanonicalLength
	for i, v := range canon {
		if v == 0 {
			cut = i
			break
		}
	}
	human := string(canon[:cut])
	return human, CostHuman, nil
}

func NewMockAPI() *wvm.GoAPI {
	return &wvm.GoAPI{
		HumanAddress:     MockHumanAddress,
		CanonicalAddress: MockCanonicalAddress,
	}
}

func TestMockApi(t *testing.T) {
	human := "foobar"
	canon, cost, err := MockCanonicalAddress(human)
	require.NoError(t, err)
	assert.Equal(t, CanonicalLength, len(canon))
	assert.Equal(t, CostCanonical, cost)

	recover, cost, err := MockHumanAddress(canon)
	require.NoError(t, err)
	assert.Equal(t, recover, human)
	assert.Equal(t, CostHuman, cost)
}

/**** MockQuerier ****/

const DEFAULT_QUERIER_GAS_LIMIT = 1_000_000

type MockQuerier struct {
	Bank    BankQuerier
	Custom  CustomQuerier
	usedGas uint64
}

var _ wvm.Querier = &MockQuerier{}

func DefaultQuerier(contractAddr string, coins wvm.Coins) wvm.Querier {
	balances := map[string]wvm.Coins{
		contractAddr: coins,
	}
	return &MockQuerier{
		Bank:    NewBankQuerier(balances),
		Custom:  NoCustom{},
		usedGas: 0,
	}
}

func (q *MockQuerier) Query(request wvm.QueryRequest, _gasLimit uint64) ([]byte, error) {
	marshaled, err := json.Marshal(request)
	if err != nil {
		return nil, err
	}
	q.usedGas += uint64(len(marshaled))
	if request.Bank != nil {
		return q.Bank.Query(request.Bank)
	}
	if request.Custom != nil {
		return q.Custom.Query(request.Custom)
	}
	if request.Staking != nil {
		return nil, wvm.UnsupportedRequest{Kind: "staking"}
	}
	if request.Wasm != nil {
		return nil, wvm.UnsupportedRequest{Kind: "wasm"}
	}
	return nil, wvm.Unknown{}
}

func (q MockQuerier) GasConsumed() uint64 {
	return q.usedGas
}

type BankQuerier struct {
	Balances map[string]wvm.Coins
}

func NewBankQuerier(balances map[string]wvm.Coins) BankQuerier {
	bal := make(map[string]wvm.Coins, len(balances))
	for k, v := range balances {
		dst := make([]wvm.Coin, len(v))
		copy(dst, v)
		bal[k] = dst
	}
	return BankQuerier{
		Balances: bal,
	}
}

func (q BankQuerier) Query(request *wvm.BankQuery) ([]byte, error) {
	if request.Balance != nil {
		denom := request.Balance.Denom
		coin := wvm.NewCoin(0, denom)
		for _, c := range q.Balances[request.Balance.Address] {
			if c.Denom == denom {
				coin = c
			}
		}
		resp := wvm.BalanceResponse{
			Amount: coin,
		}
		return json.Marshal(resp)
	}
	if request.AllBalances != nil {
		coins := q.Balances[request.AllBalances.Address]
		resp := wvm.AllBalancesResponse{
			Amount: coins,
		}
		return json.Marshal(resp)
	}
	return nil, wvm.UnsupportedRequest{Kind: "Empty BankQuery"}
}

type CustomQuerier interface {
	Query(request json.RawMessage) ([]byte, error)
}

type NoCustom struct{}

var _ CustomQuerier = NoCustom{}

func (q NoCustom) Query(request json.RawMessage) ([]byte, error) {
	return nil, wvm.UnsupportedRequest{Kind: "custom"}
}

// ReflectCustom fulfills the requirements for testing `reflect` contract
type ReflectCustom struct{}

var _ CustomQuerier = ReflectCustom{}

type CustomQuery struct {
	Ping        *struct{}         `json:"ping,omitempty"`
	Capitalized *CapitalizedQuery `json:"capitalized,omitempty"`
}

type CapitalizedQuery struct {
	Text string `json:"text"`
}

// CustomResponse is the response for all `CustomQuery`s
type CustomResponse struct {
	Msg string `json:"msg"`
}

func (q ReflectCustom) Query(request json.RawMessage) ([]byte, error) {
	var query CustomQuery
	err := json.Unmarshal(request, &query)
	if err != nil {
		return nil, err
	}
	var resp CustomResponse
	if query.Ping != nil {
		resp.Msg = "PONG"
	} else if query.Capitalized != nil {
		resp.Msg = strings.ToUpper(query.Capitalized.Text)
	} else {
		return nil, errors.New("unsupported query")
	}
	return json.Marshal(resp)
}

//************ test code for mocks *************************//

func TestBankQuerierAllBalances(t *testing.T) {
	addr := "foobar"
	balance := wvm.Coins{wvm.NewCoin(12345678, "ATOM"), wvm.NewCoin(54321, "ETH")}
	q := DefaultQuerier(addr, balance)

	// query existing account
	req := wvm.QueryRequest{
		Bank: &wvm.BankQuery{
			AllBalances: &wvm.AllBalancesQuery{
				Address: addr,
			},
		},
	}
	res, err := q.Query(req, DEFAULT_QUERIER_GAS_LIMIT)
	require.NoError(t, err)
	var resp wvm.AllBalancesResponse
	err = json.Unmarshal(res, &resp)
	require.NoError(t, err)
	assert.Equal(t, resp.Amount, balance)

	// query missing account
	req2 := wvm.QueryRequest{
		Bank: &wvm.BankQuery{
			AllBalances: &wvm.AllBalancesQuery{
				Address: "someone-else",
			},
		},
	}
	res, err = q.Query(req2, DEFAULT_QUERIER_GAS_LIMIT)
	require.NoError(t, err)
	var resp2 wvm.AllBalancesResponse
	err = json.Unmarshal(res, &resp2)
	require.NoError(t, err)
	assert.Nil(t, resp2.Amount)
}

func TestBankQuerierBalance(t *testing.T) {
	addr := "foobar"
	balance := wvm.Coins{wvm.NewCoin(12345678, "ATOM"), wvm.NewCoin(54321, "ETH")}
	q := DefaultQuerier(addr, balance)

	// query existing account with matching denom
	req := wvm.QueryRequest{
		Bank: &wvm.BankQuery{
			Balance: &wvm.BalanceQuery{
				Address: addr,
				Denom:   "ATOM",
			},
		},
	}
	res, err := q.Query(req, DEFAULT_QUERIER_GAS_LIMIT)
	require.NoError(t, err)
	var resp wvm.BalanceResponse
	err = json.Unmarshal(res, &resp)
	require.NoError(t, err)
	assert.Equal(t, resp.Amount, wvm.NewCoin(12345678, "ATOM"))

	// query existing account with missing denom
	req2 := wvm.QueryRequest{
		Bank: &wvm.BankQuery{
			Balance: &wvm.BalanceQuery{
				Address: addr,
				Denom:   "BTC",
			},
		},
	}
	res, err = q.Query(req2, DEFAULT_QUERIER_GAS_LIMIT)
	require.NoError(t, err)
	var resp2 wvm.BalanceResponse
	err = json.Unmarshal(res, &resp2)
	require.NoError(t, err)
	assert.Equal(t, resp2.Amount, wvm.NewCoin(0, "BTC"))

	// query missing account
	req3 := wvm.QueryRequest{
		Bank: &wvm.BankQuery{
			Balance: &wvm.BalanceQuery{
				Address: "someone-else",
				Denom:   "ATOM",
			},
		},
	}
	res, err = q.Query(req3, DEFAULT_QUERIER_GAS_LIMIT)
	require.NoError(t, err)
	var resp3 wvm.BalanceResponse
	err = json.Unmarshal(res, &resp3)
	require.NoError(t, err)
	assert.Equal(t, resp3.Amount, wvm.NewCoin(0, "ATOM"))
}

func TestReflectCustomQuerier(t *testing.T) {
	q := ReflectCustom{}

	// try ping
	msg, err := json.Marshal(CustomQuery{Ping: &struct{}{}})
	require.NoError(t, err)
	bz, err := q.Query(msg)
	require.NoError(t, err)
	var resp CustomResponse
	err = json.Unmarshal(bz, &resp)
	require.NoError(t, err)
	assert.Equal(t, resp.Msg, "PONG")

	// try capital
	msg2, err := json.Marshal(CustomQuery{Capitalized: &CapitalizedQuery{Text: "small."}})
	require.NoError(t, err)
	bz, err = q.Query(msg2)
	require.NoError(t, err)
	var resp2 CustomResponse
	err = json.Unmarshal(bz, &resp2)
	require.NoError(t, err)
	assert.Equal(t, resp2.Msg, "SMALL.")
}

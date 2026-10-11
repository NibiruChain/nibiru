package baseapp_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/require"

	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp"
	baseapptestutil "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/baseapp/testutil"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/telemetry"
	"github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/testutil/testdata"
	sdk "github.com/NibiruChain/nibiru/v2/lib/cosmos-sdk/types"
)

func configureExecutionTelemetry(t *testing.T, enabled bool) *prometheus.Registry {
	t.Helper()
	oldRegisterer, oldGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	reg := prometheus.NewRegistry()
	prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
	_, err := telemetry.New(telemetry.Config{Enabled: enabled, PrometheusRetentionTime: 600})
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := telemetry.New(telemetry.Config{})
		require.NoError(t, err)
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = oldRegisterer, oldGatherer
	})
	return reg
}

func executionMetric(t *testing.T, reg *prometheus.Registry, name string, labels map[string]string) *dto.Metric {
	t.Helper()
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, metric := range family.Metric {
			matches := 0
			for _, label := range metric.Label {
				if value, ok := labels[label.GetName()]; ok && value == label.GetValue() {
					matches++
				}
			}
			if matches == len(labels) {
				return metric
			}
		}
	}
	return nil
}

func TestExecutionTelemetryDelivery(t *testing.T) {
	for _, failure := range []string{"", "ante", "post", "handler", "gas", "malformed", "validation"} {
		t.Run("failure="+failure, func(t *testing.T) {
			var baselineResponse abci.ResponseDeliverTx
			var baselineHash []byte
			for _, enabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("enabled=%t", enabled), func(t *testing.T) {
					reg := configureExecutionTelemetry(t, enabled)
					suite := NewBaseAppSuite(t, func(app *baseapp.BaseApp) {
						app.SetAnteHandler(func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) {
							ctx = ctx.WithGasMeter(sdk.NewGasMeter(100000))
							if failure == "gas" {
								ctx.GasMeter().ConsumeGas(100001, "telemetry test")
							}
							if failure == "ante" {
								return ctx, errors.New("ante rejected")
							}
							return ctx, nil
						})
						if failure == "post" {
							app.SetPostHandler(func(ctx sdk.Context, _ sdk.Tx, _, _ bool) (sdk.Context, error) {
								return ctx, errors.New("post rejected")
							})
						}
					})
					baseapptestutil.RegisterCounterServer(suite.baseApp.MsgServiceRouter(), CounterServerImpl{t, capKey1, []byte("counter")})
					suite.baseApp.InitChain(abci.RequestInitChain{ConsensusParams: &tmproto.ConsensusParams{}})
					suite.baseApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
					tx := newTxCounter(t, suite.txConfig, 0, 0, 1)
					msgs := tx.GetMsgs()
					if failure == "handler" {
						msgs[1].(*baseapptestutil.MsgCounter).FailOnHandler = true
					}
					if failure == "validation" {
						msgs[0].(*baseapptestutil.MsgCounter).Counter = -1
					}
					builder := suite.txConfig.NewTxBuilder()
					require.NoError(t, builder.SetMsgs(msgs...))
					builder.SetMemo(tx.GetMemo())
					setTxSignature(t, builder, 0)
					tx = builder.GetTx()
					bz, err := suite.txConfig.TxEncoder()(tx)
					require.NoError(t, err)
					if failure == "malformed" {
						bz = []byte{0xff}
					}
					response := suite.baseApp.DeliverTx(abci.RequestDeliverTx{Tx: bz})
					require.Equal(t, failure == "", response.IsOK(), response.Log)
					suite.baseApp.EndBlock(abci.RequestEndBlock{Height: 1})
					hash := suite.baseApp.Commit().Data
					if !enabled {
						baselineResponse, baselineHash = response, hash
						return
					}
					require.Equal(t, baselineResponse, response)
					require.Equal(t, baselineHash, hash, "telemetry must not affect committed state")
					outcome := "failed"
					if failure == "" {
						outcome = "successful"
					}
					metric := executionMetric(t, reg, "nibiru_sdk_tx_exec_duration", map[string]string{"result": outcome})
					require.NotNil(t, metric)
					require.EqualValues(t, 1, metric.Histogram.GetSampleCount())
					message := executionMetric(t, reg, "nibiru_sdk_tx_msgs_total", map[string]string{"tx_result": outcome, "message_type": "/MsgCounter"})
					if failure == "malformed" {
						require.Nil(t, message)
					} else {
						require.NotNil(t, message)
						require.Equal(t, float64(2), message.Counter.GetValue())
					}
				})
			}
		})
	}
}

func TestExecutionTelemetryExcludesOtherModes(t *testing.T) {
	reg := configureExecutionTelemetry(t, true)
	suite := NewBaseAppSuite(t)
	baseapptestutil.RegisterCounterServer(suite.baseApp.MsgServiceRouter(), NoopCounterServerImpl{})
	suite.baseApp.InitChain(abci.RequestInitChain{ConsensusParams: &tmproto.ConsensusParams{}})
	suite.baseApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
	bz, err := suite.txConfig.TxEncoder()(newTxCounter(t, suite.txConfig, 0, 0))
	require.NoError(t, err)
	for _, mode := range []abci.CheckTxType{abci.CheckTxType_New, abci.CheckTxType_Recheck} {
		require.True(t, suite.baseApp.CheckTx(abci.RequestCheckTx{Tx: bz, Type: mode}).IsOK())
	}
	_, _, err = suite.baseApp.Simulate(bz)
	require.NoError(t, err)
	suite.baseApp.PrepareProposal(abci.RequestPrepareProposal{Height: 1, MaxTxBytes: 10000, Txs: [][]byte{bz}})
	suite.baseApp.ProcessProposal(abci.RequestProcessProposal{Height: 1, Txs: [][]byte{bz}})
	require.NoError(t, suite.baseApp.PrepareProposalVerifyTxBytes(bz))
	_, err = suite.baseApp.ProcessProposalVerifyTx(bz)
	require.NoError(t, err)
	require.Nil(t, executionMetric(t, reg, "nibiru_sdk_tx_exec_duration", nil))
	require.Nil(t, executionMetric(t, reg, "nibiru_sdk_tx_msgs_total", nil))
}

func TestExecutionTelemetryQueries(t *testing.T) {
	reg := configureExecutionTelemetry(t, true)
	suite := NewBaseAppSuite(t, func(app *baseapp.BaseApp) {
		testdata.RegisterQueryServer(app.GRPCQueryRouter(), testdata.QueryImpl{})
	})
	suite.baseApp.InitChain(abci.RequestInitChain{ConsensusParams: &tmproto.ConsensusParams{}})
	suite.baseApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
	suite.baseApp.EndBlock(abci.RequestEndBlock{Height: 1})
	suite.baseApp.Commit()
	req, err := (&testdata.SayHelloRequest{Name: "metrics"}).Marshal()
	require.NoError(t, err)
	response := suite.baseApp.Query(abci.RequestQuery{Path: "/testpb.Query/SayHello", Data: req})
	require.True(t, response.IsOK(), response.Log)
	require.True(t, suite.baseApp.Query(abci.RequestQuery{Path: "/app/version"}).IsOK())
	for i := 0; i < 20; i++ {
		require.True(t, suite.baseApp.Query(abci.RequestQuery{Path: fmt.Sprintf("/unknown/%d", i)}).IsErr())
		require.True(t, suite.baseApp.Query(abci.RequestQuery{Path: fmt.Sprintf("/store/unknown%d/key", i)}).IsErr())
	}
	for route, count := range map[string]uint64{"/testpb.Query/SayHello": 1, "/app": 1, "unknown": 20, "/store": 20} {
		outcome := "successful"
		if route == "unknown" || route == "/store" {
			outcome = "failed"
		}
		metric := executionMetric(t, reg, "nibiru_sdk_query_duration", map[string]string{"route": route, "result": outcome})
		require.NotNil(t, metric)
		require.Equal(t, count, metric.Histogram.GetSampleCount())
	}
	families, err := reg.Gather()
	require.NoError(t, err)
	for _, family := range families {
		require.NotContains(t, family.GetName(), "unknown19")
		if family.GetName() == "nibiru_sdk_query_duration" {
			require.Len(t, family.Metric, 4)
		}
	}
}

type telemetryPanicQuery struct{ testdata.QueryImpl }

func (telemetryPanicQuery) SayHello(context.Context, *testdata.SayHelloRequest) (*testdata.SayHelloResponse, error) {
	panic("telemetry query recovery")
}

func TestExecutionTelemetryQueryRecovery(t *testing.T) {
	reg := configureExecutionTelemetry(t, true)
	suite := NewBaseAppSuite(t, func(app *baseapp.BaseApp) {
		testdata.RegisterQueryServer(app.GRPCQueryRouter(), telemetryPanicQuery{})
	})
	suite.baseApp.InitChain(abci.RequestInitChain{ConsensusParams: &tmproto.ConsensusParams{}})
	suite.baseApp.BeginBlock(abci.RequestBeginBlock{Header: tmproto.Header{Height: 1}})
	suite.baseApp.EndBlock(abci.RequestEndBlock{Height: 1})
	suite.baseApp.Commit()
	request, err := (&testdata.SayHelloRequest{Name: "panic"}).Marshal()
	require.NoError(t, err)
	response := suite.baseApp.Query(abci.RequestQuery{Path: "/testpb.Query/SayHello", Data: request})
	require.True(t, response.IsErr())
	metric := executionMetric(t, reg, "nibiru_sdk_query_duration", map[string]string{"route": "/testpb.Query/SayHello", "result": "failed"})
	require.NotNil(t, metric)
	require.EqualValues(t, 1, metric.Histogram.GetSampleCount())
}

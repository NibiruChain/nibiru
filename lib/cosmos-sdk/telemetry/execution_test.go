package telemetry

import (
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
)

func TestExecutionMetricsConfiguration(t *testing.T) {
	oldRegisterer, oldGatherer := prometheus.DefaultRegisterer, prometheus.DefaultGatherer
	oldExecution := execution.Load()
	t.Cleanup(func() {
		prometheus.DefaultRegisterer, prometheus.DefaultGatherer = oldRegisterer, oldGatherer
		execution.Store(oldExecution)
	})
	for _, tc := range []struct {
		name string
		cfg  Config
		want bool
	}{
		{"disabled", Config{}, false},
		{"in memory only", Config{Enabled: true}, false},
		{"prometheus", Config{Enabled: true, PrometheusRetentionTime: 600, GlobalLabels: [][]string{{"chain_id", "test"}}}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			reg := prometheus.NewRegistry()
			prometheus.DefaultRegisterer, prometheus.DefaultGatherer = reg, reg
			m, err := New(tc.cfg)
			require.NoError(t, err)
			require.Equal(t, tc.want, ExecutionMetricsEnabled())
			start := time.Now().Add(-250 * time.Millisecond)
			ObserveTxExecution(start, true)
			CountTxMessage("/test.Msg", true)
			CountTxMessage("/test.Msg", true)
			CountTxMessage("/test.Msg", false)
			ObserveQuery(start, "/test.Query/Method", false)
			families, err := reg.Gather()
			require.NoError(t, err)
			if !tc.want {
				require.Empty(t, families)
				return
			}
			require.Len(t, families, 3)
			for _, family := range families {
				for _, metric := range family.Metric {
					require.Contains(t, metric.Label[0].String(), "chain_id")
					if metric.Histogram != nil {
						require.Contains(t, family.GetHelp(), "seconds")
						require.EqualValues(t, 1, metric.Histogram.GetSampleCount())
						require.InDelta(t, 0.25, metric.Histogram.GetSampleSum(), 0.1)
					}
				}
			}
			gr, err := m.Gather(FormatPrometheus)
			require.NoError(t, err)
			require.Contains(t, string(gr.Metrics), `nibiru_sdk_tx_msgs_total{chain_id="test",message_type="/test.Msg",tx_result="successful"} 2`)
			require.NotContains(t, string(gr.Metrics), "duration_seconds")
		})
	}
}

func TestExecutionMetricsRegistrationRollback(t *testing.T) {
	reg := prometheus.NewRegistry()
	existing, err := newExecutionMetrics(prometheus.NewRegistry(), nil)
	require.NoError(t, err)
	reg.MustRegister(existing.query)
	_, err = newExecutionMetrics(reg, nil)
	require.Error(t, err)
	// Earlier collectors must not remain registered after the last one fails.
	reg.Unregister(existing.query)
	_, err = newExecutionMetrics(reg, nil)
	require.NoError(t, err)
}

func BenchmarkExecutionMetrics(b *testing.B) {
	previous := execution.Load()
	b.Cleanup(func() { execution.Store(previous) })
	for _, enabled := range []bool{false, true} {
		name := "disabled"
		if enabled {
			name = "enabled"
		}
		b.Run(name, func(b *testing.B) {
			execution.Store(nil)
			if enabled {
				m, err := newExecutionMetrics(prometheus.NewRegistry(), nil)
				if err != nil {
					b.Fatal(err)
				}
				execution.Store(m)
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if ExecutionMetricsEnabled() {
					start := time.Now()
					CountTxMessage("/cosmos.bank.v1beta1.MsgSend", true)
					ObserveTxExecution(start, true)
					ObserveQuery(start, "/cosmos.bank.v1beta1.Query/Balance", true)
				}
			}
		})
	}
}

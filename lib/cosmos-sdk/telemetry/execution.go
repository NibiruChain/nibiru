package telemetry

import (
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// executionMetrics shares CometBFT's default registry and scrape endpoint.
// Unlike the legacy summaries, these histograms retain cumulative buckets.
type executionMetrics struct {
	tx       *prometheus.HistogramVec
	messages *prometheus.CounterVec
	query    *prometheus.HistogramVec
}

var execution atomic.Pointer[executionMetrics]

func newExecutionMetrics(reg prometheus.Registerer, labels prometheus.Labels) (*executionMetrics, error) {
	buckets := []float64{0.0001, 0.0005, 0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}
	m := &executionMetrics{
		tx: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "nibiru_sdk_tx_exec_duration", Help: "Application transaction execution duration in seconds, during delivery only.",
			Buckets: buckets, ConstLabels: labels,
		}, []string{"result"}),
		messages: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "nibiru_sdk_tx_msgs_total", Help: "Top-level messages in decoded delivered transactions, including failed transactions.",
			ConstLabels: labels,
		}, []string{"message_type", "tx_result"}),
		query: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "nibiru_sdk_query_duration", Help: "ABCI application query handling duration in seconds, excluding network transport.",
			Buckets: buckets, ConstLabels: labels,
		}, []string{"route", "result"}),
	}
	var registered []prometheus.Collector
	for _, collector := range []prometheus.Collector{m.tx, m.messages, m.query} {
		if err := reg.Register(collector); err != nil {
			for _, previous := range registered {
				reg.Unregister(previous)
			}
			return nil, err
		}
		registered = append(registered, collector)
	}
	return m, nil
}

// ExecutionMetricsEnabled reports whether the Prometheus execution collectors
// are active. Callers can skip timing and message inspection when disabled.
func ExecutionMetricsEnabled() bool { return execution.Load() != nil }

func executionResult(success bool) string {
	if success {
		return "successful"
	}
	return "failed"
}

// ObserveTxExecution records elapsed application execution time in seconds.
// Call after delivery recovery and gas accounting determine the final outcome.
func ObserveTxExecution(start time.Time, success bool) {
	if m := execution.Load(); m != nil {
		m.tx.WithLabelValues(executionResult(success)).Observe(time.Since(start).Seconds())
	}
}

// CountTxMessage counts one top-level message with the containing transaction's
// final outcome. typeURL must come from a decoded SDK message, never raw input.
func CountTxMessage(typeURL string, success bool) {
	if m := execution.Load(); m != nil {
		m.messages.WithLabelValues(typeURL, executionResult(success)).Inc()
	}
}

// ObserveQuery records ABCI query handling time in seconds. route must be a
// registered query method or a bounded category, never an arbitrary input path.
func ObserveQuery(start time.Time, route string, success bool) {
	if m := execution.Load(); m != nil {
		m.query.WithLabelValues(route, executionResult(success)).Observe(time.Since(start).Seconds())
	}
}

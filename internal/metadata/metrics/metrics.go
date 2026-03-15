// Package metrics defines Prometheus metrics emitted by the metadata service.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rohanyadav1024/dfs/internal/constants"
)

var (
	registerOnce sync.Once

	heartbeatsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: constants.MetricNameHeartbeatsTotal,
		Help: "Total number of node heartbeats received by metad.",
	})

	repairAttemptsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: constants.MetricNameRepairAttempts,
		Help: "Total number of repair attempts started by metad.",
	})

	repairFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: constants.MetricNameRepairFailures,
		Help: "Total number of failed repair attempts.",
	})

	repairSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: constants.MetricNameRepairSuccess,
		Help: "Total number of successful repair attempts.",
	})

	healthyNodes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: constants.MetricNameMetadataHealthy,
		Help: "Current number of healthy storage nodes.",
	})
)

// Register registers all metad metrics exactly once.
func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			heartbeatsTotal,
			repairAttemptsTotal,
			repairFailuresTotal,
			repairSuccessTotal,
			healthyNodes,
		)
	})
}

// IncHeartbeats increments the heartbeat counter.
func IncHeartbeats() {
	heartbeatsTotal.Inc()
}

// IncRepairAttempts increments the repair-attempt counter.
func IncRepairAttempts() {
	repairAttemptsTotal.Inc()
}

// IncRepairFailures increments the repair-failure counter.
func IncRepairFailures() {
	repairFailuresTotal.Inc()
}

// IncRepairSuccess increments the successful-repair counter.
func IncRepairSuccess() {
	repairSuccessTotal.Inc()
}

// SetHealthyNodes updates the current healthy-node gauge.
func SetHealthyNodes(n int) {
	healthyNodes.Set(float64(n))
}

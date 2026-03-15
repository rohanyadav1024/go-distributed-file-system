package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce sync.Once

	heartbeatsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_heartbeats_total",
		Help: "Total number of node heartbeats received by metad.",
	})

	repairAttemptsTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_repair_attempts_total",
		Help: "Total number of repair attempts started by metad.",
	})

	repairFailuresTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_repair_failures_total",
		Help: "Total number of failed repair attempts.",
	})

	repairSuccessTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_repair_success_total",
		Help: "Total number of successful repair attempts.",
	})

	healthyNodes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dfs_healthy_nodes",
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

func IncHeartbeats() {
	heartbeatsTotal.Inc()
}

func IncRepairAttempts() {
	repairAttemptsTotal.Inc()
}

func IncRepairFailures() {
	repairFailuresTotal.Inc()
}

func IncRepairSuccess() {
	repairSuccessTotal.Inc()
}

func SetHealthyNodes(n int) {
	healthyNodes.Set(float64(n))
}

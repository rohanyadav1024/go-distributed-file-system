// Package metrics defines Prometheus metrics emitted by storage nodes.
package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerOnce sync.Once

	chunksStoredTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_chunks_stored_total",
		Help: "Total number of chunks stored by storaged.",
	})

	chunksServedTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "dfs_chunks_served_total",
		Help: "Total number of chunks served by storaged.",
	})

	availableBytes = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "dfs_available_bytes",
		Help: "Current available storage bytes on this storaged node.",
	})
)

// Register registers storaged metrics exactly once.
func Register() {
	registerOnce.Do(func() {
		prometheus.MustRegister(
			chunksStoredTotal,
			chunksServedTotal,
			availableBytes,
		)
	})
}

// IncChunksStored increments the stored-chunks counter.
func IncChunksStored() {
	chunksStoredTotal.Inc()
}

// IncChunksServed increments the served-chunks counter.
func IncChunksServed() {
	chunksServedTotal.Inc()
}

// SetAvailableBytes updates the available-bytes gauge.
func SetAvailableBytes(n int64) {
	availableBytes.Set(float64(n))
}

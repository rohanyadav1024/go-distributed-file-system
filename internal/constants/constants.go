// Package constants centralizes shared literal values used across services.
package constants

import "time"

const (
	MetadataServicePrefix = "/metadata.MetadataService/"
	NodeServicePrefix     = "/node.NodeService/"
)

const (
	EnvJWTSecret         = "DFS_JWT_SECRET"
	EnvMetadataDBPath    = "DFS_METADATA_DB_PATH"
	EnvReplicationFactor = "DFS_REPLICATION_FACTOR"
	EnvStorageNodeID     = "DFS_STORAGE_NODE_ID"
	EnvStorageListenAddr = "DFS_STORAGE_LISTEN_ADDR"
	EnvMetadataAddr      = "DFS_METADATA_ADDR"
)

const (
	DefaultHeartbeatInterval   = 3 * time.Second
	DefaultNodeTimeout         = 5 * time.Second
	DefaultRepairScanInterval  = 5 * time.Second
	DefaultMetricsPollInterval = 5 * time.Second
)

const (
	MetricNamespaceDFS = "dfs"

	MetricSubsystemGRPC    = "grpc"
	MetricSubsystemCluster = "cluster"

	MetricNameRequestDurationSeconds = "request_duration_seconds"
	MetricNameHealthyNodes           = "healthy_nodes"
	MetricNameTotalNodes             = "total_nodes"
	MetricNameTotalChunks            = "total_chunks"
	MetricNameTotalReplicas          = "total_replicas"

	MetricNameHeartbeatsTotal = "dfs_heartbeats_total"
	MetricNameRepairAttempts  = "dfs_repair_attempts_total"
	MetricNameRepairFailures  = "dfs_repair_failures_total"
	MetricNameRepairSuccess   = "dfs_repair_success_total"
	MetricNameMetadataHealthy = "dfs_healthy_nodes"
	MetricLabelMethod         = "method"
)

const (
	TableFiles         = "files"
	TableChunks        = "chunks"
	TableChunkLocation = "chunk_locations"
	TableNodes         = "nodes"
	TableUploadSession = "upload_sessions"
)

const (
	NodeStatusHealthy = "healthy"
	NodeStatusDown    = "down"

	FileStatusPending   = "pending"
	FileStatusCommitted = "committed"
	FileStatusDeleted   = "deleted"

	UploadStatusPreparing = "preparing"
	UploadStatusCommitted = "committed"
)

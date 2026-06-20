package redisx

import obs "github.com/ZoneCNH/observex/pkg/observex"

// MetricClient* constants are aliases to observex shared constants so all
// foundation x-modules report identical cross-component metric names.
const (
MetricClientCreatedTotal           = obs.MetricClientCreatedTotal
MetricClientClosedTotal            = obs.MetricClientClosedTotal
MetricClientErrorsTotal            = obs.MetricClientErrorsTotal
MetricClientHealthStatus           = obs.MetricClientHealthStatus
MetricClientHealthLatencyMS        = obs.MetricClientHealthLatencyMS
MetricClientRequestsTotal          = obs.MetricClientRequestsTotal
MetricClientRequestDurationSeconds = obs.MetricClientRequestDurationSeconds
MetricClientRetriesTotal           = obs.MetricClientRetriesTotal
MetricClientInflight               = obs.MetricClientInflight

// redisx-specific metric names.
MetricRedisOperationsTotal          = "redisx_operations_total"
MetricRedisOperationDurationSeconds = "redisx_operation_duration_seconds"
MetricRedisErrorsTotal              = "redisx_errors_total"
MetricRedisPoolConnections          = "redisx_pool_connections"
MetricRedisHealthStatus             = "redisx_health_status"
)

// Metrics is the observability hook interface for redisx clients.
// It is a 3-method subset of observex.Metrics; any observex.Metrics
// implementation satisfies this interface.
type Metrics interface {
IncCounter(name string, labels map[string]string)
ObserveHistogram(name string, value float64, labels map[string]string)
SetGauge(name string, value float64, labels map[string]string)
}

// NoopMetrics is an alias for observex.NoopMetrics; it satisfies Metrics and
// discards all observations. Aliasing removes the duplicate no-op body.
type NoopMetrics = obs.NoopMetrics

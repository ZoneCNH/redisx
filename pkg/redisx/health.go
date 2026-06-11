package redisx

import (
	"context"
	"time"
)

type HealthStatusValue string

const (
	HealthHealthy   HealthStatusValue = "healthy"
	HealthDegraded  HealthStatusValue = "degraded"
	HealthUnhealthy HealthStatusValue = "unhealthy"
)

type HealthStatus struct {
	Name       string            `json:"name"`
	Component  string            `json:"component,omitempty"`
	Status     HealthStatusValue `json:"status"`
	Message    string            `json:"message,omitempty"`
	ErrorClass string            `json:"error_class,omitempty"`
	CheckedAt  time.Time         `json:"checked_at"`
	LatencyMs  int64             `json:"latency_ms"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

func (c *Client) Health(ctx context.Context) HealthStatus {
	start := time.Now()
	name := "redisx"
	var metrics Metrics
	var redisProvider Provider
	initialized := false
	closed := true
	var timeout time.Duration

	if c != nil {
		c.mu.Lock()
		name = c.cfg.Name
		metrics = c.metrics
		redisProvider = c.provider
		initialized = c.initialized
		closed = c.closed
		timeout = c.cfg.Timeout
		c.mu.Unlock()
		if name == "" {
			name = "redisx"
		}
	}

	status := func(value HealthStatusValue, message string, kind ErrorKind, metadata map[string]string) HealthStatus {
		errorClass := ""
		if kind != "" {
			errorClass = string(kind)
		}
		return HealthStatus{
			Name:       name,
			Component:  "redis",
			Status:     value,
			Message:    message,
			ErrorClass: errorClass,
			CheckedAt:  time.Now(),
			LatencyMs:  time.Since(start).Milliseconds(),
			Metadata:   metadata,
		}
	}

	if ctx == nil {
		result := status(HealthUnhealthy, "context is required", ErrorKindValidation, nil)
		recordHealthMetric(metrics, result)
		return result
	}

	if err := ctx.Err(); err != nil {
		wrapped := contextError("redisx.Health", err)
		result := status(HealthUnhealthy, err.Error(), wrapped.Kind, nil)
		recordHealthMetric(metrics, result)
		return result
	}

	if !initialized {
		result := status(HealthUnhealthy, "client is not initialized", ErrorKindValidation, nil)
		recordHealthMetric(metrics, result)
		return result
	}

	if closed {
		result := status(HealthUnhealthy, "client is closed", ErrorKindClosed, nil)
		recordHealthMetric(metrics, result)
		return result
	}

	if timeout > 0 {
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline)
			if remaining <= 0 {
				message := context.DeadlineExceeded.Error()
				if err := ctx.Err(); err != nil {
					message = err.Error()
				}
				result := status(HealthUnhealthy, message, ErrorKindTimeout, nil)
				recordHealthMetric(metrics, result)
				return result
			}
			if remaining < timeout {
				result := status(HealthDegraded, "context deadline is shorter than client timeout", ErrorKindTimeout, map[string]string{
					"reason":  "deadline_below_timeout",
					"timeout": timeout.String(),
				})
				recordHealthMetric(metrics, result)
				return result
			}
		}
	}

	if redisProvider != nil {
		if err := redisProvider.Ping(ctx); err != nil {
			wrapped := providerError("redisx.Health", err)
			result := status(HealthUnhealthy, wrapped.Message, wrapped.Kind, nil)
			recordHealthMetric(metrics, result)
			return result
		}
	}

	result := status(HealthHealthy, "ok", "", nil)
	recordHealthMetric(metrics, result)
	return result
}

func (c *Client) HealthCheck(ctx context.Context) HealthStatus {
	return c.Health(ctx)
}

func recordHealthMetric(metrics Metrics, status HealthStatus) {
	if metrics == nil {
		return
	}
	labels := map[string]string{
		"name":   status.Name,
		"status": string(status.Status),
	}
	metrics.SetGauge(MetricClientHealthStatus, healthGaugeValue(status.Status), labels)
	metrics.ObserveHistogram(MetricClientHealthLatencyMS, float64(status.LatencyMs), labels)
	metrics.SetGauge(MetricRedisHealthStatus, healthGaugeValue(status.Status), labels)
}

func healthGaugeValue(status HealthStatusValue) float64 {
	if status == HealthHealthy {
		return 1
	}
	return 0
}

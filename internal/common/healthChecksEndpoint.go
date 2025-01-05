package common

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sawyerwatts/world-one/internal/common/middleware"
)

type HealthStatus string

// HealthStatusHealthy indicates that the app or health check are running as
// expected.
const HealthStatusHealthy HealthStatus = "Healthy"

// HealthStatusDegrated indicates that the app or health check are experiencing
// issues but may be degraded (as opposed to unhealthy/inoperable).
const HealthStatusDegraded HealthStatus = "Degraded"

// HealthStatusHealthy indicates that the app or health check cannot run in any
// capacity.
const HealthStatusUnhealthy HealthStatus = "Unhealthy"

type HealthCheck struct {
	Name string
	// Check is permitted to panic, that will be caught by the caller.
	Check func(c *gin.Context, slogger *slog.Logger) HealthCheckResult
}

type HealthCheckResult struct {
	Status  HealthStatus
	Payload map[string]any
}

func NewHealthChecksEndpoint(healthChecks []HealthCheck) func(c *gin.Context) {
	type check struct {
		Name     string         `json:"name"`
		Status   string         `json:"status"`
		Duration string         `json:"duration"`
		Payload  map[string]any `json:"payloadDict"`
	}
	type healthCheckOverview struct {
		Status   string  `json:"status"`
		Duration string  `json:"duration"`
		Checks   []check `json:"checks"`
	}

	execAndRecover := func(
		c *gin.Context,
		slogger *slog.Logger,
		check func(c *gin.Context, slogger *slog.Logger) HealthCheckResult,
	) (result HealthCheckResult) {
		defer func() {
			if r := recover(); r != nil {
				result = HealthCheckResult{
					Status:  HealthStatusUnhealthy,
					Payload: map[string]any{"panic": r},
				}
			}
		}()
		return check(c, slogger)
	}

	return func(c *gin.Context) {
		slogger := middleware.MustGetSlogger(c)

		overview := healthCheckOverview{
			Checks: make([]check, 0, len(healthChecks)),
		}

		overviewStart := time.Now()
		for _, individualHealthCheck := range healthChecks {
			check := check{}
			checkStart := time.Now()
			result := execAndRecover(c, slogger, individualHealthCheck.Check)
			checkEnd := time.Now()

			check.Name = individualHealthCheck.Name
			check.Status = string(result.Status)
			check.Duration = checkEnd.Sub(checkStart).String()
			check.Payload = result.Payload
			overview.Checks = append(overview.Checks, check)
		}
		overviewEnd := time.Now()
		overview.Duration = overviewEnd.Sub(overviewStart).String()

		overview.Status = string(HealthStatusHealthy)
		if len(healthChecks) == 0 {
			c.JSON(http.StatusOK, overview)
			return
		}
		allUnhealthy := true
		for _, check := range overview.Checks {
			if check.Status != string(HealthStatusHealthy) {
				overview.Status = string(HealthStatusDegraded)
				slogger.ErrorContext(c, "A health check did not come back healthy", slog.String("name", check.Name), slog.Any("payload", check.Payload))
			}
			if check.Status != string(HealthStatusUnhealthy) {
				allUnhealthy = false
			}
		}
		if allUnhealthy {
			overview.Status = string(HealthStatusUnhealthy)
		}

		c.JSON(http.StatusOK, overview)
	}
}

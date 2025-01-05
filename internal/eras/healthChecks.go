package eras

import (
	"log/slog"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/db"
)

func AppendHealthChecks(checks []common.HealthCheck, dbPool *pgxpool.Pool) []common.HealthCheck {
	return append(checks,
		common.HealthCheck{
			Name: "Assert current era exists",
			Check: func(c *gin.Context, slogger *slog.Logger) common.HealthCheckResult {
				dbQueries := db.New(dbPool)
				eraQueries := MakeQueries(dbQueries, slogger)
				currEra, err := eraQueries.GetCurrEra(c)
				if err != nil {
					return common.HealthCheckResult{
						Status:  common.HealthStatusUnhealthy,
						Payload: map[string]any{"err": err.Error()},
					}
				}
				return common.HealthCheckResult{
					Status:  common.HealthStatusHealthy,
					Payload: map[string]any{"currEra": currEra},
				}
			},
		})
}

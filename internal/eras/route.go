package eras

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: but what if need slogger from middleware? get from ctx?

func Route(
	v1 *gin.RouterGroup,
	dbConnString string,
	slogger *slog.Logger,
) {
	group := v1.Group("/eras")

	group.GET("", func(c *gin.Context) {
		dbConn, err := pgx.Connect(c, dbConnString)
		if err != nil {
			slogger.Error("Could not connect to DB", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "Could not connect to DB")
		}
		defer dbConn.Close(c)

		dbQueries := db.New(dbConn)
		eraQueries := MakeQueries(dbQueries, slogger)
		allEras, err := eraQueries.GetEras(c)
		if err != nil {
			slogger.Error("An unexpected error was returned by the DB integration", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, fmt.Sprintf("An unexpected error was returned by the DB integration"))
		}

		eraDTOs := make([]EraDTO, 0, len(allEras))
		for _, era := range allEras {
			eraDTO := MakeEraDTO(era)
			eraDTOs = append(eraDTOs, eraDTO)
		}

		c.JSON(http.StatusOK, eraDTOs)
	})

	group.GET("/current", func(c *gin.Context) {
		dbConn, err := pgx.Connect(c, dbConnString)
		if err != nil {
			slogger.Error("Could not connect to DB", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "Could not connect to DB")
		}
		defer dbConn.Close(c)

		dbQueries := db.New(dbConn)
		eraQueries := MakeQueries(dbQueries, slogger)
		era, err := eraQueries.GetCurrEra(c)
		if err != nil {
			if errors.Is(err, ErrNoCurrEra) {
				c.String(http.StatusInternalServerError, "There is no current era, the game is not initialized yet")
				return
			}
			slogger.Error("An unexpected error was returned by the DB integration", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, fmt.Sprintf("An unexpected error was returned by the DB integration"))
			return
		}

		c.JSON(http.StatusOK, MakeEraDTO(era))
	})
}

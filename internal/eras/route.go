package eras

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sawyerwatts/world-one/internal/common/middleware"
	"github.com/sawyerwatts/world-one/internal/db"
)

func Route(
	v1 *gin.RouterGroup,
	dbPool *pgxpool.Pool,
) {
	group := v1.Group("/eras")

	group.GET("", func(c *gin.Context) {
		slogger := middleware.MustGetSlogger(c)
		dbQueries := db.New(dbPool)
		eraQueries := MakeQueries(dbQueries, slogger)
		allEras, err := eraQueries.GetEras(c)
		if err != nil {
			slogger.ErrorContext(c, "An unexpected error was returned by the DB integration", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned by the DB integration")
		}

		eraDTOs := make([]EraDTO, len(allEras))
		for i, era := range allEras {
			eraDTOs[i] = MakeEraDTO(era)
		}

		c.JSON(http.StatusOK, eraDTOs)
	})

	group.GET("/current", func(c *gin.Context) {
		slogger := middleware.MustGetSlogger(c)
		dbQueries := db.New(dbPool)
		eraQueries := MakeQueries(dbQueries, slogger)
		era, err := eraQueries.GetCurrEra(c)
		if err != nil {
			if errors.Is(err, ErrNoCurrEra) {
				slogger.ErrorContext(c, "There is no current era, the game is not initialized yet")
				c.String(http.StatusInternalServerError, "There is no current era, the game is not initialized yet")
				return
			}
			slogger.ErrorContext(c, "An unexpected error was returned by the DB integration", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned by the DB integration")
			return
		}

		c.JSON(http.StatusOK, MakeEraDTO(era))
	})

	group.POST("/rollover", func(c *gin.Context) {
		slogger := middleware.MustGetSlogger(c)
		newEraName := c.Query("newEraName")
		if len(newEraName) == 0 {
			c.String(http.StatusBadRequest, "Expected query parameter newEraName but not given or was empty")
			return
		}

		tx, err := dbPool.BeginTx(c, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			slogger.ErrorContext(c, "Could not begin serializable transaciton", slog.String("err", err.Error()))
		}
		defer func() {
			err := tx.Rollback(c)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slogger.ErrorContext(c, "Transaction rollback failed unexpectedly", slog.String("err", err.Error()))
				c.String(http.StatusInternalServerError, "Transaction rollback failed unexpectedly")
				return
			}
		}()

		dbQueries := db.New(tx)
		eraQueries := MakeQueries(dbQueries, slogger)

		newEra, prevEra, err := Rollover(c, eraQueries, dbQueries, slogger, time.Now().UTC(), newEraName)
		if err != nil {
			if errors.Is(err, ErrWhitespaceEraName) {
				c.String(http.StatusBadRequest, "Expected query parameter newEraName but not given or was empty")
				return
			}
			if errors.Is(err, ErrDuplicateEraName) {
				c.String(http.StatusBadRequest, "The given new era's name is a duplicate of another pre-existing era")
				return
			}
			slogger.ErrorContext(c, "An unexpected error was returned when rolling over the era(s)", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned when rolling over the era(s)")
			return
		}

		err = tx.Commit(c)
		if err != nil {
			slogger.ErrorContext(c, "An unexpected error was returned when committing the changes", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned when committing the changes")
			return
		}

		var prevEraDTO *EraDTO
		if prevEra != nil {
			p := MakeEraDTO(*prevEra)
			prevEraDTO = &p
		}
		resp := struct {
			NewEraDTO  EraDTO  `json:"newEraDTO"`
			PrevEraDTO *EraDTO `json:"prevEraDTO"`
		}{
			NewEraDTO:  MakeEraDTO(newEra),
			PrevEraDTO: prevEraDTO,
		}
		c.JSON(http.StatusCreated, resp)
	})
}

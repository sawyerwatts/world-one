package eras

import (
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sawyerwatts/world-one/internal/db"
)

// TODO: what would this look like if it was a ctlr?
// TODO: idempotent POSTs middleware
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
			c.String(http.StatusInternalServerError, "An unexpected error was returned by the DB integration")
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
				slogger.Error("There is no current era, the game is not initialized yet")
				c.String(http.StatusInternalServerError, "There is no current era, the game is not initialized yet")
				return
			}
			slogger.Error("An unexpected error was returned by the DB integration", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned by the DB integration")
			return
		}

		c.JSON(http.StatusOK, MakeEraDTO(era))
	})

	group.POST("/rollover", func(c *gin.Context) {
		newEraName := c.Query("newEraName")
		if len(newEraName) == 0 {
			c.String(http.StatusBadRequest, "Expected query parameter newEraName but not given or was empty")
			return
		}

		dbConn, err := pgx.Connect(c, dbConnString)
		if err != nil {
			slogger.Error("Could not connect to DB", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "Could not connect to DB")
		}
		defer dbConn.Close(c)

		tx, err := dbConn.BeginTx(c, pgx.TxOptions{IsoLevel: pgx.Serializable})
		if err != nil {
			slogger.Error("Could not begin serializable transaciton", slog.String("err", err.Error()))
		}
		defer func() {
			err := tx.Rollback(c)
			if err != nil && !errors.Is(err, pgx.ErrTxClosed) {
				slogger.Error("Transaction rollback failed unexpectedly", slog.String("err", err.Error()))
				c.String(http.StatusInternalServerError, "Transaction rollback failed unexpectedly")
				return
			}
		}()

		dbQueries := db.New(tx)
		eraQueries := MakeQueries(dbQueries, slogger)
		rollover := MakeRollover(eraQueries, dbQueries, slogger)

		newEra, prevEra, err := rollover.Exec(c, time.Now().UTC(), newEraName)
		if err != nil {
			if errors.Is(err, ErrWhitespaceEraName) {
				c.String(http.StatusBadRequest, "Expected query parameter newEraName but not given or was empty")
				return
			}
			slogger.Error("An unexpected error was returned when rolling over the era(s)", slog.String("err", err.Error()))
			c.String(http.StatusInternalServerError, "An unexpected error was returned when rolling over the era(s)")
			return
		}

		err = tx.Commit(c)
		if err != nil {
			slogger.Error("An unexpected error was returned when committing the changes", slog.String("err", err.Error()))
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

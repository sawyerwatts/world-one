package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/sawyerwatts/world-one/internal/db"
	"github.com/sawyerwatts/world-one/internal/eras"
)

func main() {
	// TODO: read settings via viper
	//	then update SlogIncludeSource to default to true
	mainSettings := makeMainSettings()

	ctx := context.Background()

	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: mainSettings.SlogIncludeSource}))
	slog.SetDefault(slogger)

	router := gin.Default()
	// TODO: configure Gin to use slogger
	// TODO: improve req logging

	// TODO: remove this placeholder w/ actual endpoints
	//	Don't just hardcode the endpoints+handlers in main
	//	Start openapi spec, and have an endpoint for that too
	v1 := router.Group("/v1")
	{
		erasGroup := v1.Group("/eras")
		erasGroup.GET("", func(c *gin.Context) {
			dbConn, err := pgx.Connect(ctx, mainSettings.DBConnectionString)
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Could not connect to DB: %v", err))
			}
			defer dbConn.Close(ctx)

			dbQueries := db.New(dbConn)
			erasGetter := eras.MakeGetter(dbQueries, slogger)
			allEras, err := erasGetter.GetEras(c)
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("An unexpected error was returned by the DB integration: %v", err))
			}

			eraDTOs := make([]eras.EraDTO, 0, len(allEras))
			for _, era := range allEras {
				eraDTO := eras.MakeEraDTO(era)
				eraDTOs = append(eraDTOs, eraDTO)
			}

			c.JSON(http.StatusOK, eraDTOs)
		})

		erasGroup.GET("/current", func(c *gin.Context) {
			dbConn, err := pgx.Connect(ctx, mainSettings.DBConnectionString)
			if err != nil {
				c.String(http.StatusInternalServerError, fmt.Sprintf("Could not connect to DB: %v", err))
			}
			defer dbConn.Close(ctx)

			dbQueries := db.New(dbConn)
			erasGetter := eras.MakeGetter(dbQueries, slogger)
			era, err := erasGetter.GetCurrEra(ctx)
			if err != nil {
				if errors.Is(err, eras.ErrNoCurrEra) {
					c.String(http.StatusInternalServerError, "There is no current era, the game is not initialized yet")
					return
				}
				c.String(http.StatusInternalServerError, fmt.Sprintf("An unexpected error was returned by the DB integration: %v", err))
				return
			}

			c.JSON(http.StatusOK, eras.MakeEraDTO(era))
		})
	}

	s := http.Server{
		Addr:           mainSettings.Addr,
		Handler:        router,
		ReadTimeout:    time.Duration(mainSettings.ReadTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(mainSettings.WriteTimeoutSec) * time.Second,
		IdleTimeout:    time.Duration(mainSettings.IdleTimeoutSec) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	slogger.InfoContext(ctx, "Starting HTTP server")
	exitCode := 0
	go func() {
		err := s.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			slogger.InfoContext(ctx, "Shutting down server")
		} else {
			slog.Error("An bad error was returned by shut down server", slog.String("err", err.Error()))
			exitCode = 1
		}
	}()

	slogger.InfoContext(ctx, "Send interrupt or terminate signals to start gracefully shutting down the server")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	slogger.Info("Received term or interrupt signal, will shutdown gracefully within a number of seconds", slog.Int("timeLimitSec", mainSettings.MaxGracefulShutdownSec))
	ctx, cancel := context.WithTimeout(ctx, time.Duration(mainSettings.MaxGracefulShutdownSec))
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slogger.ErrorContext(ctx, "Server errored while shutting down", slog.String("err", err.Error()))
		exitCode = 1
	}

	os.Exit(exitCode)
}

type mainSettings struct {
	Addr                   string
	ReadTimeoutSec         int
	WriteTimeoutSec        int
	IdleTimeoutSec         int
	MaxGracefulShutdownSec int
	SlogIncludeSource      bool
	DBConnectionString     string
}

func makeMainSettings() mainSettings {
	return mainSettings{
		Addr:                   "localhost:8080",
		ReadTimeoutSec:         30,
		WriteTimeoutSec:        90,
		IdleTimeoutSec:         120,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
		DBConnectionString:     "dbname=world_one",
	}
}

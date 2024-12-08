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

	_ "net/http/pprof" // BUG: how add auth to these endpoints?

	"github.com/gin-gonic/gin"
	"github.com/sawyerwatts/world-one/internal/common/middleware"
	"github.com/sawyerwatts/world-one/internal/eras"
)

// TODO: curr opr-level checklist task: adding assertions
// TODO: curr app-level checklist task: finalizing logging
//	use gin.New() instead of gin.Default()
//		update gin router to use slogger, esp w/ traceUUID
//		write own panic protection

func main() {
	mainSettings := makeMainSettings()

	loc, err := time.LoadLocation(mainSettings.TimeZone)
	if err != nil {
		panic(fmt.Sprintf("Couldn't set timezone to '%s'", mainSettings.TimeZone))
	}
	time.Local = loc

	ctx := context.Background()

	slogger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: mainSettings.SlogIncludeSource}))
	slog.SetDefault(slogger)

	router := gin.Default()

	router.Use(middleware.UseTraceUUIDAndSlogger(slogger))

	v1 := router.Group("/v1")
	eras.Route(v1, mainSettings.DBConnectionString)

	s := http.Server{
		Addr:           mainSettings.Addr,
		Handler:        router,
		ReadTimeout:    time.Duration(mainSettings.ReadTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(mainSettings.WriteTimeoutSec) * time.Second,
		IdleTimeout:    time.Duration(mainSettings.IdleTimeoutSec) * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	slogger.InfoContext(ctx, "Starting HTTP server", slog.String("addr", mainSettings.Addr))
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
	ctx, cancel := context.WithTimeout(ctx, time.Duration(mainSettings.MaxGracefulShutdownSec)*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slogger.ErrorContext(ctx, "Server errored while shutting down", slog.String("err", err.Error()))
		exitCode = 1
	}

	os.Exit(exitCode)
}

type mainSettings struct {
	TimeZone               string
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
		TimeZone:               "GMT",
		Addr:                   "localhost:8080",
		ReadTimeoutSec:         30,
		WriteTimeoutSec:        90,
		IdleTimeoutSec:         120,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
		DBConnectionString:     "dbname=world_one",
	}
}

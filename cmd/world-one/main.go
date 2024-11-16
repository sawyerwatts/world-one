package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
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
	//	Start openapi spec, and have an endpoint for that too
	router.GET("/helloWorld", func(c *gin.Context) {
		c.String(http.StatusOK, "Hello, world!")
	})

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
			exitCode += 1
		}
	}()

	slogger.InfoContext(ctx, "Send INT or TERM signals to start gracefully shutting down the server")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGTERM, syscall.SIGINT)
	<-quit
	slogger.Info("Received term or interrupt signal, will shutdown gracefully within a number of seconds", slog.Int("timeLimitSec", mainSettings.MaxGracefulShutdownSec))
	ctx, cancel := context.WithTimeout(ctx, time.Duration(mainSettings.MaxGracefulShutdownSec))
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slogger.ErrorContext(ctx, "Server errored while shutting down", slog.String("err", err.Error()))
		exitCode += 10
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
}

func makeMainSettings() mainSettings {
	return mainSettings{
		Addr:                   "localhost:8080",
		ReadTimeoutSec:         30,
		WriteTimeoutSec:        90,
		IdleTimeoutSec:         120,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
	}
}

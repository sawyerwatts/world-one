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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sawyerwatts/world-one/internal/common/middleware"
	"github.com/sawyerwatts/world-one/internal/eras"
	"github.com/spf13/viper"
)

// TODO: curr opr-level checklist task: adding assertions
// TODO: curr app-level checklist task: configs

func main() {
	ctx := context.Background()

	var mainSettings *mainSettings
	{
		// TODO: use the embed API
		v := viper.New()
		v.SetEnvPrefix("W1")
		v.BindEnv("PGURL")
		v.SetConfigFile("./cmd/world-one/config.json")
		if err := v.ReadInConfig(); err != nil {
			panic("viper failed to read configs: " + err.Error())
		}
		mainSettings = newMainSettings()
		if err := v.Unmarshal(mainSettings); err != nil {
			panic("viper failed to unmarshal configs: " + err.Error())
		}
		if err := mainSettings.Validate(); err != nil {
			panic("settings failed to validate: " + err.Error())
		}
	}

	{
		loc, err := time.LoadLocation(mainSettings.TimeZone)
		if err != nil {
			panic(fmt.Sprintf("Couldn't set timezone to '%s'", mainSettings.TimeZone))
		}
		time.Local = loc
	}

	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: mainSettings.SlogIncludeSource})
	slogger := slog.New(slogHandler)
	slog.SetDefault(slogger)

	dbPool, err := pgxpool.New(context.Background(), mainSettings.DBConnectionString)
	if err != nil {
		panic(err)
	}
	defer dbPool.Close()

	router := gin.Default()
	{
		// TODO: use gin.New() instead of gin.Default()?
		//		update gin router to use slogger, esp w/ traceUUID
		//		write own panic protection

		router.Use(middleware.UseTraceUUIDAndSlogger(ctx, slogger))

		v1 := router.Group("/v1")

		eras.Route(v1, dbPool)
	}

	s := http.Server{
		Addr:           mainSettings.Addr,
		Handler:        router,
		ReadTimeout:    time.Duration(mainSettings.ReadTimeoutSec) * time.Second,
		WriteTimeout:   time.Duration(mainSettings.WriteTimeoutSec) * time.Second,
		IdleTimeout:    time.Duration(mainSettings.IdleTimeoutSec) * time.Second,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       slog.NewLogLogger(slogHandler, slog.LevelError),
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
	slogger.InfoContext(ctx, "Received term or interrupt signal, will shutdown gracefully within a number of seconds", slog.Int("timeLimitSec", mainSettings.MaxGracefulShutdownSec))
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
	DBConnectionString     string `mapstructure:"PGURL"`
}

func newMainSettings() *mainSettings {
	return &mainSettings{
		TimeZone:               "GMT",
		Addr:                   "",
		ReadTimeoutSec:         30,
		WriteTimeoutSec:        90,
		IdleTimeoutSec:         120,
		MaxGracefulShutdownSec: 5,
		SlogIncludeSource:      false,
		DBConnectionString:     "",
	}
}

func (s *mainSettings) Validate() error {
	if s.TimeZone == "" {
		return errors.New("setting TimeZone is not initialized")
	}
	if s.Addr == "" {
		return errors.New("setting Addr is not initialized")
	}
	if s.ReadTimeoutSec < 1 {
		return errors.New("setting ReadTimeoutSec is not positive")
	}
	if s.WriteTimeoutSec < 1 {
		return errors.New("setting WriteTimeoutSec is not positive")
	}
	if s.IdleTimeoutSec < 1 {
		return errors.New("setting IdleTimeoutSec is not positive")
	}
	if s.MaxGracefulShutdownSec < 1 {
		return errors.New("setting MaxGracefulShutdownSec is not positive")
	}
	if s.DBConnectionString == "" {
		return errors.New("setting DBConnectionString is not initialized")
	}
	return nil
}

package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	_ "net/http/pprof" // BUG: how add auth to these endpoints?

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sawyerwatts/world-one/internal/common"
	"github.com/sawyerwatts/world-one/internal/common/middleware"
	"github.com/sawyerwatts/world-one/internal/eras"
)

// BUG: remember how to get pprof working (then put under the admin tag on oapi?)

// TODO: impl paging on eras
//	not because it's actually relevant there, but as an exercise

// TODO: curr opr-level checklist task: README.md/assertions (just restart)
// TODO: curr app-level checklist task: webApis.md/etags (just restart)
// TODO: review security.md after auth is implemented

func main() {
	ctx := context.Background()

	mainConfig := mustGetConfig()

	{
		loc, err := time.LoadLocation(mainConfig.TimeZone)
		if err != nil {
			panic(fmt.Sprintf("Couldn't set timezone to '%s'", mainConfig.TimeZone))
		}
		time.Local = loc
	}

	slogHandler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: mainConfig.SlogIncludeSource})
	slogger := slog.New(slogHandler)
	slog.SetDefault(slogger)

	dbPool, err := pgxpool.New(context.Background(), mainConfig.DBConnectionString)
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

		{
			checks := make([]common.HealthCheck, 0, 2)
			checks = append(checks, common.HealthCheck{
				Name: "Test DB Connectivity",
				Check: func(c *gin.Context, _ *slog.Logger) common.HealthCheckResult {
					_, err := dbPool.Exec(c, "select now()")
					if err != nil {
						return common.HealthCheckResult{
							Status:  common.HealthStatusUnhealthy,
							Payload: map[string]any{"err": err.Error()},
						}
					}
					return common.HealthCheckResult{Status: common.HealthStatusHealthy}
				},
			})
			checks = eras.AppendHealthChecks(checks, dbPool)
			router.GET("/healthChecks", common.NewHealthChecksEndpoint(checks))
		}

		v1 := router.Group("/v1")
		v1.GET("", func(c *gin.Context) {
			c.Header("Content-Type", "text/html; charset=utf-8")
			c.HTML(http.StatusOK, "scalar-v1.html", gin.H{})
		})

		eras.Route(v1, dbPool)
		router.StaticFile("/favicon.ico", filepath.Join(mainConfig.WebsiteDir, "favicon.ico"))
		router.StaticFile("/open-api-v1.yml", filepath.Join(mainConfig.WebsiteDir, "open-api-v1.yml"))
		router.LoadHTMLGlob(filepath.Join(mainConfig.WebsiteDir, "*.html"))
	}

	s := http.Server{
		Addr: mainConfig.Addr,
		Handler: http.TimeoutHandler(
			router,
			time.Duration(mainConfig.RequestTimeoutMS)*time.Millisecond,
			fmt.Sprintf("The request timed out as it ran longer than %d milliseconds", mainConfig.RequestTimeoutMS)),
		ReadHeaderTimeout: time.Duration(mainConfig.ReadHeaderTimeoutMS) * time.Millisecond,
		ReadTimeout:       time.Duration(mainConfig.ReadTimeoutMS) * time.Millisecond,
		// WriteTimeout isn't configured since it closes the conn without
		// sending a response code and doesn't propogate the context.
		IdleTimeout:    time.Duration(mainConfig.IdleTimeoutMS) * time.Millisecond,
		MaxHeaderBytes: 1 << 20,
		ErrorLog:       slog.NewLogLogger(slogHandler, slog.LevelError),
	}

	slogger.InfoContext(ctx, "Starting HTTP server", slog.String("addr", mainConfig.Addr))
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
	slogger.InfoContext(ctx, "Received term or interrupt signal, will shutdown gracefully within a number of seconds", slog.Int("timeLimitSec", mainConfig.MaxGracefulShutdownSec))
	ctx, cancel := context.WithTimeout(ctx, time.Duration(mainConfig.MaxGracefulShutdownSec)*time.Second)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slogger.ErrorContext(ctx, "Server errored while shutting down", slog.String("err", err.Error()))
		exitCode = 1
	}

	os.Exit(exitCode)
}

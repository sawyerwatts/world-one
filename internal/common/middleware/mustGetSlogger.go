package middleware

import (
	"log/slog"

	"github.com/gin-gonic/gin"
)

const sloggerContextKey = "slogger"

func MustGetSlogger(c *gin.Context) *slog.Logger {
	sloggerAny, ok := c.Get(sloggerContextKey)
	if ok {
		return sloggerAny.(*slog.Logger)
	}
	panic("Could not retrieve slogger from context")
}

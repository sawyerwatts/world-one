package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceUUIDContextKey = "traceUUID"
const TraceUUIDHeader = "X-TRACE-UUID"

// UseTraceUUIDAndSlogger is middleware that will, if present, retrieve the
// trace UUID from the request's header (else create a new one), set the
// response trace UUID header, add the trace UUID to a new slogger, and add the
// trace UUID and new slogger to the context.
//
// This is intended to be a very early middleware, so much so that it does not
// check if there is a slogger in the context.
func UseTraceUUIDAndSlogger(slogger *slog.Logger) func(c *gin.Context) {
	return func(c *gin.Context) {
		var traceUUID uuid.UUID
		var err error
		if givenUUID := c.GetHeader(TraceUUIDHeader); givenUUID != "" {
			traceUUID, err = uuid.Parse(givenUUID)
			if err != nil {
				c.String(http.StatusBadRequest, "Given a trace UUID that could not be parsed: "+err.Error())
			}
		} else {
			traceUUID, err = uuid.NewV7()
			if err != nil {
				panic("Failed to create a new trace UUID: " + err.Error())
			}
		}

		sloggerWithTraceUUID := slogger.With(slog.String("traceUUID", traceUUID.String()))
		sloggerWithTraceUUID.Info("Trace UUID has been added to this request's slogger and the context")
		c.Header(TraceUUIDHeader, traceUUID.String())
		c.Set(TraceUUIDContextKey, traceUUID)
		c.Set(sloggerContextKey, sloggerWithTraceUUID)
	}
}

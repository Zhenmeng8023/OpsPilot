package middleware

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const TraceIDKey = "traceId"

func TraceID() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader("X-Trace-Id")
		if traceID == "" {
			traceID = newTraceID()
		}
		c.Set(TraceIDKey, traceID)
		c.Header("X-Trace-Id", traceID)
		c.Next()
	}
}

func newTraceID() string {
	var buf [12]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "trc_fallback"
	}
	return "trc_" + hex.EncodeToString(buf[:])
}

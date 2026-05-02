package response

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/app/middleware"
)

type Envelope struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
	TraceID string      `json:"traceId"`
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, 0, "success", data)
}

func Fail(c *gin.Context, httpStatus, code int, message string) {
	JSON(c, httpStatus, code, message, nil)
}

func JSON(c *gin.Context, httpStatus, code int, message string, data interface{}) {
	c.JSON(httpStatus, Envelope{
		Code:    code,
		Message: message,
		Data:    data,
		TraceID: traceID(c),
	})
}

func traceID(c *gin.Context) string {
	if value, exists := c.Get(middleware.TraceIDKey); exists {
		if traceID, ok := value.(string); ok {
			return traceID
		}
	}
	return ""
}

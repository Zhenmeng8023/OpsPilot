package response

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"

	"opspilot/server/internal/app/middleware"
	"opspilot/server/internal/shared/apperror"
)

type Envelope struct {
	Code         int         `json:"code"`
	Message      string      `json:"message"`
	Data         interface{} `json:"data"`
	TraceID      string      `json:"traceId"`
	DebugMessage string      `json:"debugMessage,omitempty"`
}

func Success(c *gin.Context, data interface{}) {
	JSON(c, http.StatusOK, 0, "success", data)
}

func Fail(c *gin.Context, httpStatus, code int, message string) {
	JSONWithDebug(c, httpStatus, code, message, "", nil)
}

func FailAppError(c *gin.Context, appErr *apperror.Error) {
	if appErr == nil {
		Fail(c, http.StatusInternalServerError, 500000, "internal error")
		return
	}
	debugMessage := ""
	if appErr.Cause != nil {
		trace := traceID(c)
		slog.Error(
			"request failed",
			"traceId", trace,
			"httpStatus", appErr.HTTPStatus,
			"code", appErr.Code,
			"message", appErr.Message,
			"cause", appErr.Cause.Error(),
		)
		if debugEnabled() {
			debugMessage = appErr.Cause.Error()
		}
	}
	JSONWithDebug(c, appErr.HTTPStatus, appErr.Code, appErr.Message, debugMessage, nil)
}

func JSON(c *gin.Context, httpStatus, code int, message string, data interface{}) {
	JSONWithDebug(c, httpStatus, code, message, "", data)
}

func JSONWithDebug(c *gin.Context, httpStatus, code int, message string, debugMessage string, data interface{}) {
	c.JSON(httpStatus, Envelope{
		Code:         code,
		Message:      message,
		Data:         data,
		TraceID:      traceID(c),
		DebugMessage: strings.TrimSpace(debugMessage),
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

func debugEnabled() bool {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("APP_ENV")))
	switch env {
	case "", "local", "dev", "development", "test":
		return true
	default:
		return false
	}
}

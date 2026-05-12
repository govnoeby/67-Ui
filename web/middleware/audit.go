package middleware

import (
	"bytes"
	"io"
	"strings"

	"github.com/govnoeby/67-Ui/v3/web/service"

	"github.com/gin-gonic/gin"
)

// safeMethods is the set of HTTP methods that are not logged as actions.
var silentMethods = map[string]bool{
	"GET":    true,
	"HEAD":   true,
	"OPTIONS": true,
}

// AuditMiddleware logs all unsafe (non-GET/HEAD/OPTIONS) API requests to the audit log.
func AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if silentMethods[c.Request.Method] {
			c.Next()
			return
		}

		// Skip public endpoints
		path := c.Request.URL.Path
		if strings.Contains(path, "/login") ||
			strings.Contains(path, "/csrf-token") ||
			strings.Contains(path, "/swagger/") {
			c.Next()
			return
		}

		// Read body for detail
		bodyBytes := []byte{}
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		detail := strings.TrimSpace(string(bodyBytes))
		if len(detail) > 500 {
			detail = detail[:500] + "..."
		}

		action := methodToAction(c.Request.Method) + " " + strings.TrimPrefix(path, "/panel/")

		// Wrap the response writer to capture status
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		svc := service.AuditLogService{}
		svc.Record(c, action, detail)
	}
}

func methodToAction(method string) string {
	switch method {
	case "POST":
		return "CREATE"
	case "PUT":
		return "UPDATE"
	case "DELETE":
		return "DELETE"
	case "PATCH":
		return "PATCH"
	default:
		return method
	}
}

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

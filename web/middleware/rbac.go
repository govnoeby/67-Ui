package middleware

import (
	"net/http"

	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/govnoeby/67-Ui/v3/web/session"

	"github.com/gin-gonic/gin"
)

// RequireRole returns a middleware that checks the authenticated user has
// at least one of the specified roles. If no roles are specified, any
// authenticated user passes.
func RequireRole(roles ...model.Role) gin.HandlerFunc {
	return func(c *gin.Context) {
		user := session.GetLoginUser(c)
		if user == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		if len(roles) == 0 {
			c.Next()
			return
		}
		for _, role := range roles {
			if user.Role == role {
				c.Next()
				return
			}
		}
		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"success": false,
			"msg":     "insufficient permissions",
		})
	}
}

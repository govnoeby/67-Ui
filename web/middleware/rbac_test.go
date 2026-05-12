package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/gin-gonic/gin"
)

func setTestUser(c *gin.Context, role model.Role) {
	c.Set("api_auth_user", &model.User{
		Id:       1,
		Username: "testuser",
		Role:     role,
		IsActive: true,
	})
}

func TestRequireRoleMiddleware_Authenticated(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		userRole   model.Role
		required   []model.Role
		wantStatus int
	}{
		{
			name:       "admin can access admin endpoint",
			userRole:   model.RoleAdmin,
			required:   []model.Role{model.RoleAdmin},
			wantStatus: http.StatusOK,
		},
		{
			name:       "operator can access operator endpoint",
			userRole:   model.RoleOperator,
			required:   []model.Role{model.RoleOperator},
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer can access viewer endpoint",
			userRole:   model.RoleViewer,
			required:   []model.Role{model.RoleViewer},
			wantStatus: http.StatusOK,
		},
		{
			name:       "viewer denied admin endpoint",
			userRole:   model.RoleViewer,
			required:   []model.Role{model.RoleAdmin},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "operator denied admin endpoint",
			userRole:   model.RoleOperator,
			required:   []model.Role{model.RoleAdmin},
			wantStatus: http.StatusForbidden,
		},
		{
			name:       "empty required roles allows all",
			userRole:   model.RoleViewer,
			required:   []model.Role{},
			wantStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			setTestUser(c, tt.userRole)

			handler := RequireRole(tt.required...)
			handler(c)

			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestRequireRole_GetUserFromAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	user := &model.User{Id: 1, Username: "test", Role: model.RoleAdmin, IsActive: true}
	c.Set("api_auth_user", user)

	handler := RequireRole(model.RoleAdmin)
	handler(c)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
}

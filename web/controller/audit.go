package controller

import (
	"strconv"

	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/govnoeby/67-Ui/v3/web/middleware"
	"github.com/govnoeby/67-Ui/v3/web/service"

	"github.com/gin-gonic/gin"
)

// AuditController handles audit log viewing.
type AuditController struct {
	BaseController
	auditService service.AuditLogService
}

func NewAuditController(g *gin.RouterGroup) *AuditController {
	a := &AuditController{}
	a.initRouter(g)
	return a
}

func (a *AuditController) initRouter(g *gin.RouterGroup) {
	g = g.Group("/audit")
	g.Use(middleware.RequireRole(model.RoleAdmin, model.RoleOperator))
	g.GET("/logs", a.getLogs)
}

func (a *AuditController) getLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "50"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 200 {
		pageSize = 50
	}

	logs, total, err := a.auditService.GetLogs(page, pageSize)
	if err != nil {
		jsonMsg(c, "failed to get audit logs", err)
		return
	}
	jsonObj(c, gin.H{
		"logs":  logs,
		"total": total,
		"page":  page,
	}, nil)
}

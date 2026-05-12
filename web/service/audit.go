package service

import (
	"time"

	"github.com/govnoeby/67-Ui/v3/database"
	"github.com/govnoeby/67-Ui/v3/database/model"
	"github.com/govnoeby/67-Ui/v3/web/session"

	"github.com/gin-gonic/gin"
)

const auditLogRetentionDays = 90

// AuditLogService provides methods to record and query audit log entries.
type AuditLogService struct{}

func (s *AuditLogService) Record(c *gin.Context, action, detail string) {
	user := session.GetLoginUser(c)
	userID := 0
	username := "anonymous"
	if user != nil {
		userID = user.Id
		username = user.Username
	}

	entry := &model.AuditLog{
		UserId:   userID,
		Username: username,
		Action:   action,
		Path:     c.Request.URL.Path,
		Method:   c.Request.Method,
		IP:       c.ClientIP(),
		Detail:   detail,
		Status:   c.Writer.Status(),
	}

	db := database.GetDB()
	db.Create(entry)

	// Prune old logs periodically (1% chance per write)
	if entry.Id%100 == 0 {
		go s.pruneOld()
	}
}

func (s *AuditLogService) GetLogs(page, pageSize int) ([]model.AuditLog, int64, error) {
	db := database.GetDB()
	var logs []model.AuditLog
	var total int64

	db.Model(&model.AuditLog{}).Count(&total)

	offset := (page - 1) * pageSize
	err := db.Model(&model.AuditLog{}).
		Order("created_at desc").
		Offset(offset).
		Limit(pageSize).
		Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}

func (s *AuditLogService) pruneOld() {
	db := database.GetDB()
	cutoff := time.Now().AddDate(0, 0, -auditLogRetentionDays)
	db.Where("created_at < ?", cutoff).Delete(&model.AuditLog{})
}

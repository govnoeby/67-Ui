package service

import (
	"testing"

	"github.com/govnoeby/67-Ui/v3/database/model"
)

func TestAuditLogModel(t *testing.T) {
	// Verify that AuditLog struct has the required fields
	log := model.AuditLog{
		UserId:   1,
		Username: "admin",
		Action:   "CREATE inbound",
		Path:     "/panel/api/inbounds",
		Method:   "POST",
		IP:       "127.0.0.1",
		Detail:   "created inbound test",
		Status:   200,
	}

	if log.UserId != 1 {
		t.Errorf("expected UserId 1, got %d", log.UserId)
	}
	if log.Username != "admin" {
		t.Errorf("expected Username admin, got %s", log.Username)
	}
	if log.Action != "CREATE inbound" {
		t.Errorf("expected Action 'CREATE inbound', got %s", log.Action)
	}
	if log.Status != 200 {
		t.Errorf("expected Status 200, got %d", log.Status)
	}
}

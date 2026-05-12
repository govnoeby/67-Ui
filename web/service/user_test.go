package service

import (
	"testing"

	"github.com/govnoeby/67-Ui/v3/database/model"
)

func TestUserRoles(t *testing.T) {
	cases := []struct {
		role model.Role
		want bool
	}{
		{model.RoleAdmin, true},
		{model.RoleOperator, true},
		{model.RoleViewer, true},
		{model.Role("superadmin"), false},
		{model.Role(""), false},
		{model.Role("user"), false},
	}

	for _, tc := range cases {
		if got := tc.role.IsValid(); got != tc.want {
			t.Errorf("Role(%q).IsValid() = %v, want %v", tc.role, got, tc.want)
		}
	}
}

func TestUserRoleHierarchy(t *testing.T) {
	// Verify that roles are distinct
	if model.RoleAdmin == model.RoleOperator {
		t.Error("admin and operator roles must be distinct")
	}
	if model.RoleOperator == model.RoleViewer {
		t.Error("operator and viewer roles must be distinct")
	}
}

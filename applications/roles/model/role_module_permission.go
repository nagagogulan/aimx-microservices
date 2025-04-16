package model

import "time"

type RoleModulePermission struct {
	ID        uint      `json:"id"`
	RoleID    uint      `json:"role_id"`
	ModuleID  uint      `json:"module_id"`
	PermissionID uint   `json:"permission_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (RoleModulePermission) TableName() string {
	return "role_module_permission"
}

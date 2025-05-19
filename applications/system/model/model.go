package model

import (
	"github.com/PecozQ/aimx-library/domain/dto"
)

type GetAuditLogResponse struct {
	AuditLogs  []dto.AuditLogs `json:"audit_logs"`
	PagingInfo PagingInfo      `json:"paging_info"`
}
type PagingInfo struct {
	TotalItems  int64 `json:"total_items"`
	CurrentPage int   `json:"current_page"`
	TotalPage   int   `json:"total_page"`
	ItemPerPage int   `json:"item_per_page"`
}

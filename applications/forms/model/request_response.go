package model

import (
	entity "github.com/PecozQ/aimx-library/domain/entities"
)

type CreateTemplateRequest struct {
	Template entity.Template `json:"template"`
}

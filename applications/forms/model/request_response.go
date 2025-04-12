package model

import (
	entity "github.com/PecozQ/aimx-library/domain/entities"
)

// type CreateTemplateRequest struct {
// 	Template entity.Template `json:"template"`
// }

//	type TemplateRequest struct {
//		ID string `json:"id"`
//	}
type TemplateRequest struct {
	ID       string          `json:"id"`       // ID of the template to be updated
	Template entity.Template `json:"template"` // Updated template payload
}
type Template struct {
	Template entity.Template `json:"template"` // Updated template payload
}
type ParamRequest struct {
	ID   string `json:"id"`
	Type int    `json:"type"`
}
type Response struct {
	Message string `json:"message"`
}

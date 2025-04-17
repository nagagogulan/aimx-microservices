package model

import (
	entity "github.com/PecozQ/aimx-library/domain/entities"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// type CreateTemplateRequest struct {
// 	Template entity.Template `json:"template"`
// }

//	type TemplateRequest struct {
//		ID string `json:"id"`
//	}
type TemplateRequest struct {
	ID       string          `json:"id"`
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
type FormType struct {
	ID   primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Name string             `bson:"name" json:"name"`
}
type FormTypeResponse struct {
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
}

package model

import (
	"time"

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

type FormDTO struct {
	ID             primitive.ObjectID     `json:"_id"`
	OrganizationID string                 `json:"organization_id"`
	Status         int                    `json:"status"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
	Type           int                    `json:"type"`
	Sections       []Section              `json:"sections"`
	Fields         map[string]interface{} `json:"fields"`
}

type Section struct {
	ID       int    `json:"id"`
	Label    string `json:"label"`
	Position int    `json:"position"`
}

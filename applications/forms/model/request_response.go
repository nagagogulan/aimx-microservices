package model

import (
	"context"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
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
	ID       string `json:"id"`
	Status   int    `json:"status"`
	Type     int    `json:"type"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
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
	ID             primitive.ObjectID     `json:"id"`
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

// type GetFormResponse struct {
// 	FormDtoData []*FormDTO `json:"formdtoData"`
// 	PagingInfo  PagingInfo `json:"pagingInfo"`
// }

type PagingInfo struct {
	TotalItems  int64 `json:"total_items"`
	CurrentPage int   `json:"current_page"`
	TotalPage   int   `json:"total_page"`
	ItemPerPage int   `json:"item_per_page"`
}
type GetFormResponse struct {
	FormDtoData []map[string]interface{} `json:"formdtoData"`
	PagingInfo  PagingInfo               `json:"pagingInfo"`
}
type SearchFormsRequest struct {
	Type        int             `json:"type"`
	Status      int             `json:"status"`
	SearchParam dto.SearchParam `json:"searchparam"`
	Ctx         context.Context `json:"-"`
}

type SearchFormsResponse struct {
	Forms []dto.FormDTO `json:"forms"`
	Total int64         `json:"total"`
}
type SearchFormsByOrganizationRequest struct {
	FormName string `json:"formname"`
	Type     int    `json:"type"`
	Page     int    `json:"page"`
	PageSize int    `json:"pageSize"`
}
type SearchFormByNamesResponse struct {
	Form *dto.FormDTO `json:"form"`
}
type ContextKey string
type SearchParam struct {
	Page   int            `json:"page"`
	Size   int            `json:"size"`
	Filter []Filterfields `json:"filter"`
}

type Filterfields struct {
	Fields string `json:"fields"` // e.g. "Organization Name"
	Value  string `json:"value"`  // e.g. "ORG"
}
type CreateFormRequestWithCtx struct {
	Ctx  context.Context
	Form *dto.FormDTO
}
type GetAllDocketDetailsResponse struct {
	Data  []entity.ModelConfig `json:"docketMetrics,omitempty"`
	Error string               `json:"error,omitempty"`
}

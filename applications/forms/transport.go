package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	errorlib "github.com/PecozQ/aimx-library/apperrors"
	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"

	//"github.com/gorilla/mux"
	"whatsdare.com/fullstack/aimx/backend/model"
)

func MakeHttpHandler(s service.Service) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errorlib.EncodeError)}

	r := gin.New()
	endpoints := NewEndpoint(s)

	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Next()
	})

	router := r.Group(fmt.Sprintf("%s/%s", commonlib.BasePath, commonlib.Version))

	//Register and Login Endpoints...
	router.POST("/template/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateTemplateEndpoint,
		decodeCreateTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))
	// Get Template by ID
	router.GET("/template", gin.WrapF(httptransport.NewServer(
		endpoints.GetTemplateByIDEndpoint,
		decodeGetTemplateByTypeRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Update Template
	router.PUT("/template/update", gin.WrapF(httptransport.NewServer(
		endpoints.UpdateTemplateEndpoint,
		decodeUpdateTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Delete Template
	router.DELETE("/template", gin.WrapF(httptransport.NewServer(
		endpoints.DeleteTemplateEndpoint,
		decodeDeleteTemplateRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	router.POST("/form/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateFormEndpoint,
		decodeCreateFormRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Get Template by ID
	router.GET("/form", gin.WrapF(httptransport.NewServer(
		endpoints.GetFormByTypeEndpoint, // ✅ changed from GetTemplateByTypeEndpoint
		decodeGetFormByTypeRequest,      // ✅ updated to match GetTemplateByID
		encodeResponse,
		options...,
	).ServeHTTP))
	router.POST("/formtype/create", gin.WrapF(httptransport.NewServer(
		endpoints.CreateFormTypeEndpoint,
		decodeCreateFormTypeRequest,
		encodeResponse,
		options...,
	).ServeHTTP))

	// Get Template by ID
	router.GET("/formtype/list", gin.WrapF(httptransport.NewServer(
		endpoints.GetFormTypeEndpoint, // ✅ changed from GetTemplateByTypeEndpoint
		decodeGetFormTypeRequest,      // ✅ updated to match GetTemplateByID
		encodeResponse,
		options...,
	).ServeHTTP))

	return r
}

// Decode register api request...
func decodeCreateTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {

	var request entities.Template
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}

// Decode register api request...
func decodeCreateFormRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.TemplateDTO
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	// newCtx, err := commonlib.ExtractGinContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	return &dto.FormDTO{Type: request.Type, Sections: request.Sections, Fields: request.Fields}, nil
}
func decodeCreateFormTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request dto.FormType
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	// newCtx, err := commonlib.ExtractGinContext(ctx)
	// if err != nil {
	// 	return nil, err
	// }
	return &request, nil
}
func decodeGetTemplateByTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id := strings.TrimSpace(r.URL.Query().Get("id")) // remove quotes if passed in URL
	typeStr := strings.TrimSpace(r.URL.Query().Get("type"))

	req := &model.ParamRequest{}

	if id != "" {
		req.ID = id
	}

	if typeStr != "" {

		req.Type = typeStr
	}

	if req.ID == "" && req.Type == "" {
		return nil, fmt.Errorf("either 'id' or 'type' must be provided")
	}

	return req, nil
}

func decodeGetFormByTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	// query := r.URL.Query()
	// idStr := query.Get("id")
	// typeStr := query.Get("type")

	// if typeStr == "" {
	// 	return nil, fmt.Errorf("missing 'type' query parameter")
	// }

	// typeInt, err := strconv.Atoi(typeStr)
	// if err != nil {
	// 	return nil, fmt.Errorf("invalid 'type' query parameter: %v", err)
	// }
	// vars := mux.Vars(r)     // get path params
	// typeStr := vars["type"]
	// id := strings.Trim(r.URL.Query().Get("id"), `"`) // remove quotes if passed in URL
	typeStr := r.URL.Query().Get("type")
	req := &model.ParamRequest{}

	// if id != "" {
	// 	req.ID = id
	// }

	if typeStr != "" {

		req.Type = typeStr
	}

	if req.ID == "" && req.Type == "" {
		return nil, fmt.Errorf("either 'id' or 'type' must be provided")
	}

	return req, nil
}
func decodeGetFormTypeRequest(ctx context.Context, r *http.Request) (interface{}, error) {

	return nil, nil
}
func decodeUpdateTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request entities.Template

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	return request, nil
}
func decodeDeleteTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	id := strings.TrimSpace(r.URL.Query().Get("id")) // remove quotes if passed in URL
	req := &model.ParamRequest{}

	if id != "" {
		req.ID = id
	}
	return req, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

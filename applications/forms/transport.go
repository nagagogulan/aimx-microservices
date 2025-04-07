package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	commonlib "github.com/PecozQ/aimx-library/common"
	errorlib "github.com/PecozQ/aimx-library/errors"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"whatsdare.com/fullstack/aimx/backend/service"

	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
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

	return r
}

// Decode register api request...
func decodeCreateTemplateRequest(ctx context.Context, r *http.Request) (interface{}, error) {
	var request model.CreateTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		return nil, err
	}
	// Extract Gin context
	newCtx, err := commonlib.ExtractGinContext(ctx)
	if err != nil {
		return nil, err
	}
	return middleware.RequestWithContext{Ctx: newCtx, Request: request}, nil
}

func encodeResponse(ctx context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	return json.NewEncoder(w).Encode(response)
}

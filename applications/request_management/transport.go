package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gofrs/uuid"
)

func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	r := gin.New()
	r.Use(gin.Recovery())

	api := r.Group(fmt.Sprintf("%s/%s/request", common.BasePath, common.Version))
	{
		api.POST("/", gin.WrapF(httptransport.NewServer(
			endpoints.CreateRequestEndpoint,
			decodeCreateRequest,
			encodeResponse,
		).ServeHTTP))

		api.PUT("/status", gin.WrapF(httptransport.NewServer(
			endpoints.UpdateRequestStatusEndpoint,
			decodeUpdateRequestStatus,
			encodeResponse,
		).ServeHTTP))

		api.GET("/org", gin.WrapF(httptransport.NewServer(
			endpoints.GetRequestsByOrgEndpoint,
			decodeOrgIDFromQuery,
			encodeResponse,
		).ServeHTTP))

		api.GET("/", gin.WrapF(httptransport.NewServer(
			endpoints.GetAllRequestsEndpoint,
			decodeEmptyRequest,
			encodeResponse,
		).ServeHTTP))

		api.GET("/:id", gin.WrapF(httptransport.NewServer(
			endpoints.GetRequestByIDEndpoint,
			decodeIDFromPath,
			encodeResponse,
		).ServeHTTP))
	}

	return r
}

func decodeCreateRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.CreateRequestDTO
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateRequestStatus(_ context.Context, r *http.Request) (interface{}, error) {
	fmt.Println("called transport")
	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		return nil, fmt.Errorf("missing id parameter")
	}

	id, err := uuid.FromString(idStr)
	if err != nil {
		return nil, fmt.Errorf("invalid UUID format for id: %w", err)
	}

	var dto dto.UpdateRequestStatusDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		return nil, err
	}
	return map[string]interface{}{"id": id, "dto": &dto}, nil
}

func decodeOrgIDFromQuery(_ context.Context, r *http.Request) (interface{}, error) {
	idStr := r.URL.Query().Get("org_id")
	return uuid.FromString(idStr)
}

func decodeEmptyRequest(_ context.Context, r *http.Request) (interface{}, error) {
	return nil, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func decodeIDFromPath(_ context.Context, r *http.Request) (interface{}, error) {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	return uuid.FromString(idStr)
}
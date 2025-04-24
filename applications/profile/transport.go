package base

import (
	"context"
	"encoding/json"
	"net/http"
	"fmt"
	"strings"


	"github.com/gin-gonic/gin"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/common"


)



func MakeHTTPHandler(endpoints Endpoints) http.Handler {
	r := gin.New()

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	// Role
	api := router.Group("/profile")
	// r := gin.Default()
	// endpoints := NewEndpoint(s)

	// api := r.Group("/api/v1/profile")
	{
		api.GET("/:id", gin.WrapF(httptransport.NewServer(
            endpoints.GetUserProfileEndpoint,
            decodeUUIDParam, // This will now extract 'id' from the URL path
            encodeResponse,
        ).ServeHTTP))

        // PUT: /profile
        api.PUT("/", gin.WrapF(httptransport.NewServer(
            endpoints.UpdateUserProfileEndpoint,
            decodeUpdateUserRequest, // Decode body to user
            encodeResponse,
        ).ServeHTTP))
	}

	return r
}


func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
	// This assumes path ends with /:id
	parts := strings.Split(r.URL.Path, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid path")
	}
	idStr := parts[len(parts)-1]
	return idStr, nil // ← string is passed to endpoint
}


func decodeUpdateUserRequest(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdateUserRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		return nil, err
	}
	
	// Convert to entity.User
	user := &entities.User{
		ID:       req.ID,
		FullName: req.FullName,
		UserName: req.UserName,
		Country:  req.Country,
	}
	
	return user, nil
}

func encodeResponse(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

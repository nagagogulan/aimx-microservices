package base

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gofrs/uuid"
	"github.com/PecozQ/aimx-library/common"

	"github.com/PecozQ/aimx-library/domain/dto"
)

func MakeRoleHTTPHandler(endpoints RoleEndpoints) http.Handler {

	r := gin.New()

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s", common.BasePath, common.Version))

	// Role
	role := router.Group("/roles")
	role.POST("/", wrapEndpoint(endpoints.CreateRole, decodeCreateRole, encode))
	role.GET("/:id", wrapEndpoint(endpoints.GetRoleByID, decodeUUIDParam, encode))
	role.PUT("/", wrapEndpoint(endpoints.UpdateRole, decodeUpdateRole, encode))
	role.DELETE("/:id", wrapEndpoint(endpoints.DeleteRole, decodeUUIDParam, encode))
	role.GET("/", wrapEndpoint(endpoints.ListRoles, decodeEmpty, encode))

	// Module
	module := router.Group("/modules")
	module.POST("/", wrapEndpoint(endpoints.CreateModule, decodeCreateModule, encode))
	module.GET("/:id", wrapEndpoint(endpoints.GetModuleByID, decodeUUIDParam, encode))
	module.PUT("/", wrapEndpoint(endpoints.UpdateModule, decodeUpdateModule, encode))
	module.DELETE("/:id", wrapEndpoint(endpoints.DeleteModule, decodeUUIDParam, encode))
	module.GET("/", wrapEndpoint(endpoints.ListModules, decodeEmpty, encode))

	// Permission
	perm := router.Group("/permissions")
	perm.POST("/", wrapEndpoint(endpoints.CreatePermission, decodeCreatePermission, encode))
	perm.GET("/:id", wrapEndpoint(endpoints.GetPermissionByID, decodeUUIDParam, encode))
	perm.PUT("/", wrapEndpoint(endpoints.UpdatePermission, decodeUpdatePermission, encode))
	perm.DELETE("/:id", wrapEndpoint(endpoints.DeletePermission, decodeUUIDParam, encode))
	perm.GET("/", wrapEndpoint(endpoints.ListPermissions, decodeEmpty, encode))

	// RMP
	rmp := router.Group("/rolePermission")
	rmp.POST("/", wrapEndpoint(endpoints.CreateRMP, decodeCreateRMP, encode))
	rmp.GET("/:id", wrapEndpoint(endpoints.GetRMPByID, decodeUUIDParam, encode))
	rmp.PUT("/", wrapEndpoint(endpoints.UpdateRMP, decodeUpdateRMP, encode))
	rmp.DELETE("/:id", wrapEndpoint(endpoints.DeleteRMP, decodeUUIDParam, encode))
	rmp.GET("/", wrapEndpoint(endpoints.ListRMPs, decodeEmpty, encode))

	roleDetails := router.Group("/getRoleDetails")
	roleDetails.GET("/:roleID", wrapEndpoint(endpoints.GetModulesAndPermissionsByRoleID, decodeUUIDParam, encode))


	return r
}

// --- Endpoint Wrapper --- //
func wrapEndpoint(ep endpoint.Endpoint, decoder httptransport.DecodeRequestFunc, encoder httptransport.EncodeResponseFunc) gin.HandlerFunc {
	return gin.WrapF(httptransport.NewServer(ep, decoder, encoder).ServeHTTP)
}

// --- Decoders --- //

func decodeEmpty(_ context.Context, _ *http.Request) (interface{}, error) {
	return nil, nil
}

func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	return uuid.FromString(idStr)
}

func decodeCreateRole(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateRole(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeCreateModule(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.CreateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateModule(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeCreatePermission(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdatePermission(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeCreateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.CreateRMPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	var req dto.UpdateRMPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

// --- Encoder --- //
func encode(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}


// func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
// 	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
// 	return uuid.FromString(idStr)
// }
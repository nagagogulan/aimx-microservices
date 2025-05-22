package base

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	middleware "github.com/PecozQ/aimx-library/middleware"
	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/endpoint"
	httptransport "github.com/go-kit/kit/transport/http"
	"github.com/gofrs/uuid"
)

func MakeRoleHTTPHandler(endpoints RoleEndpoints) http.Handler {
	options := []httptransport.ServerOption{httptransport.ServerErrorEncoder(errcom.EncodeError)}
	r := gin.New()
	r.Use(gin.Logger())

	// r.Use(cors.New(cors.Config{
	// 	AllowOrigins:     []string{"http://54.251.96.179:3000", "http://localhost:3000"}, // Replace with your frontend's origin
	// 	AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
	// 	AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
	// 	AllowCredentials: true,
	// }))

	// Base router group: /api/v1
	router := r.Group(fmt.Sprintf("%s/%s/%s", common.BasePath, common.Version, "role"))

	// Role
	// Role
	// role := router.Group("/roles")
	router.POST("/", wrapEndpoint(endpoints.CreateRole, decodeCreateRole, encode, options))
	router.GET("/:id", wrapEndpoint(endpoints.GetRoleByID, decodeUUIDParam, encode, options))
	router.PUT("/", wrapEndpoint(endpoints.UpdateRole, decodeUpdateRole, encode, options))
	router.DELETE("/:id", wrapEndpoint(endpoints.DeleteRole, decodeUUIDParam, encode, options))
	router.GET("/", wrapEndpoint(endpoints.ListRoles, decodeEmpty, encode, options))
	router.GET("/test", wrapEndpoint(endpoints.TestKong, decodeEmpty, encode, options))

	// Module
	module := router.Group("/modules")
	module.POST("/", wrapEndpoint(endpoints.CreateModule, decodeCreateModule, encode, options))
	module.GET("/:id", wrapEndpoint(endpoints.GetModuleByID, decodeUUIDParam, encode, options))
	module.PUT("/", wrapEndpoint(endpoints.UpdateModule, decodeUpdateModule, encode, options))
	module.DELETE("/:id", wrapEndpoint(endpoints.DeleteModule, decodeUUIDParam, encode, options))
	module.GET("/", wrapEndpoint(endpoints.ListModules, decodeEmpty, encode, options))

	// Permission
	perm := router.Group("/permissions")
	perm.POST("/", wrapEndpoint(endpoints.CreatePermission, decodeCreatePermission, encode, options))
	perm.GET("/:id", wrapEndpoint(endpoints.GetPermissionByID, decodeUUIDParam, encode, options))
	perm.PUT("/", wrapEndpoint(endpoints.UpdatePermission, decodeUpdatePermission, encode, options))
	perm.DELETE("/:id", wrapEndpoint(endpoints.DeletePermission, decodeUUIDParam, encode, options))
	perm.GET("/", wrapEndpoint(endpoints.ListPermissions, decodeEmpty, encode, options))

	// RMP
	rmp := router.Group("/rolePermission")
	rmp.POST("/", wrapEndpoint(endpoints.CreateRMP, decodeCreateRMP, encode, options))
	rmp.GET("/:id", wrapEndpoint(endpoints.GetRMPByID, decodeUUIDParam, encode, options))
	rmp.PUT("/", wrapEndpoint(endpoints.UpdateRMP, decodeUpdateRMP, encode, options))
	rmp.DELETE("/:id", wrapEndpoint(endpoints.DeleteRMP, decodeUUIDParam, encode, options))
	rmp.GET("/", wrapEndpoint(endpoints.ListRMPs, decodeEmpty, encode, options))
	rmp.POST("/bulk", wrapEndpoint(endpoints.FlexibleBulkCreateRMP, decodeFlexibleCreateRMP, encode, options))
	rmp.POST("/bulk-update", wrapEndpoint(endpoints.FlexibleBulkUpdateRMP, decodeFlexibleBulkUpdateRMP, encode, options))

	// Role Details
	roleDetails := router.Group("/getRoleDetails")
	roleDetails.GET("/:roleID", wrapEndpoint(endpoints.GetModulesAndPermissionsByRoleID, decodeUUIDParam, encode, options))
	return r
}

// --- Endpoint Wrapper --- //
func wrapEndpoint(ep endpoint.Endpoint, decoder httptransport.DecodeRequestFunc, encoder httptransport.EncodeResponseFunc, option []httptransport.ServerOption) gin.HandlerFunc {
	return gin.WrapF(httptransport.NewServer(ep, decoder, encoder, option...).ServeHTTP)
}

// --- Decoders --- //

func decodeEmpty(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	return nil, nil
}

func decodeUUIDParam(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	return uuid.FromString(idStr)
}

func decodeCreateRole(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.CreateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	return &req, nil
}

func decodeUpdateRole(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.UpdateRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeCreateModule(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.CreateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeUpdateModule(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.UpdateModuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	return &req, nil
}

func decodeCreatePermission(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.CreatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	return &req, nil
}

func decodeUpdatePermission(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.UpdatePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, err
	}
	return &req, nil
}

func decodeCreateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.CreateRMPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	return &req, nil
}

func decodeUpdateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var req dto.UpdateRMPRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT
	}
	return &req, nil
}

// --- Encoder --- //
func encode(_ context.Context, w http.ResponseWriter, response interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

func decodeFlexibleCreateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	roleID, err := uuid.FromString(raw["role_id"].(string))
	if err != nil {
		return nil, err
	}

	var moduleIDs []uuid.UUID
	switch v := raw["module_id"].(type) {
	case string:
		id, _ := uuid.FromString(v)
		moduleIDs = append(moduleIDs, id)
	case []interface{}:
		for _, m := range v {
			id, _ := uuid.FromString(m.(string))
			moduleIDs = append(moduleIDs, id)
		}
	default:
		return nil, fmt.Errorf("invalid module_id format")
	}

	// handle permission_id
	var flatPerms []uuid.UUID
	var mappedPerms [][]uuid.UUID

	switch rawPerms := raw["permission_id"].(type) {
	case string:
		id, _ := uuid.FromString(rawPerms)
		flatPerms = append(flatPerms, id)

	case []interface{}:
		if len(rawPerms) > 0 {
			switch rawPerms[0].(type) {
			case string:
				// flat array
				for _, item := range rawPerms {
					id, _ := uuid.FromString(item.(string))
					flatPerms = append(flatPerms, id)
				}
			case []interface{}:
				// nested permission_id (by module)
				for _, innerList := range rawPerms {
					var group []uuid.UUID
					for _, item := range innerList.([]interface{}) {
						id, _ := uuid.FromString(item.(string))
						group = append(group, id)
					}
					mappedPerms = append(mappedPerms, group)
				}
			}
		}
	default:
		return nil, fmt.Errorf("invalid permission_id format")
	}

	var finalPerm interface{}
	if len(mappedPerms) > 0 {
		finalPerm = mappedPerms
	} else {
		finalPerm = flatPerms
	}

	return &dto.FlexibleCreateRMPRequest{
		RoleID:       roleID,
		ModuleIDs:    moduleIDs,
		PermissionID: finalPerm,
	}, nil
}

func decodeFlexibleBulkUpdateRMP(_ context.Context, r *http.Request) (interface{}, error) {
	_, err := middleware.DecodeHeaderGetClaims(r)
	if err != nil {
		return nil, errcom.ErrInvalidOrMissingJWT // Unauthorized or invalid token
	}
	var raw map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
		return nil, err
	}

	roleID, err := uuid.FromString(raw["role_id"].(string))
	if err != nil {
		return nil, err
	}

	var moduleIDs []uuid.UUID
	switch v := raw["module_id"].(type) {
	case string:
		id, _ := uuid.FromString(v)
		moduleIDs = append(moduleIDs, id)
	case []interface{}:
		for _, m := range v {
			id, _ := uuid.FromString(m.(string))
			moduleIDs = append(moduleIDs, id)
		}
	default:
		return nil, fmt.Errorf("invalid module_id format")
	}

	// Handle permission_id
	var flatPerms []uuid.UUID
	var mappedPerms [][]uuid.UUID

	switch rawPerms := raw["permission_id"].(type) {
	case string:
		id, _ := uuid.FromString(rawPerms)
		flatPerms = append(flatPerms, id)
	case []interface{}:
		if len(rawPerms) > 0 {
			switch rawPerms[0].(type) {
			case string:
				// flat array
				for _, item := range rawPerms {
					id, _ := uuid.FromString(item.(string))
					flatPerms = append(flatPerms, id)
				}
			case []interface{}:
				// nested permission_id (by module)
				for _, innerList := range rawPerms {
					var group []uuid.UUID
					for _, item := range innerList.([]interface{}) {
						id, _ := uuid.FromString(item.(string))
						group = append(group, id)
					}
					mappedPerms = append(mappedPerms, group)
				}
			}
		}
	default:
		return nil, fmt.Errorf("invalid permission_id format")
	}

	var finalPerm interface{}
	if len(mappedPerms) > 0 {
		finalPerm = mappedPerms
	} else {
		finalPerm = flatPerms
	}

	return &dto.FlexibleCreateRMPRequest{
		RoleID:       roleID,
		ModuleIDs:    moduleIDs,
		PermissionID: finalPerm,
	}, nil
}

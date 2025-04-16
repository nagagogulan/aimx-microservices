package base

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"github.com/gofrs/uuid"

	"github.com/PecozQ/aimx-library/domain/dto"
	"whatsdare.com/fullstack/aimx/backend/service"
)

type RoleEndpoints struct {
	CreateRole     endpoint.Endpoint
	GetRoleByID    endpoint.Endpoint
	UpdateRole     endpoint.Endpoint
	DeleteRole     endpoint.Endpoint
	ListRoles      endpoint.Endpoint

	CreateModule   endpoint.Endpoint
	GetModuleByID  endpoint.Endpoint
	UpdateModule   endpoint.Endpoint
	DeleteModule   endpoint.Endpoint
	ListModules    endpoint.Endpoint

	CreatePermission  endpoint.Endpoint
	GetPermissionByID endpoint.Endpoint
	UpdatePermission  endpoint.Endpoint
	DeletePermission  endpoint.Endpoint
	ListPermissions   endpoint.Endpoint

	CreateRMP  endpoint.Endpoint
	GetRMPByID endpoint.Endpoint
	UpdateRMP  endpoint.Endpoint
	DeleteRMP  endpoint.Endpoint
	ListRMPs   endpoint.Endpoint
	GetModulesAndPermissionsByRoleID endpoint.Endpoint

}

func NewRoleEndpoints(
	roleService service.RoleService,
	moduleService service.ModuleService,
	permissionService service.PermissionService,
	rmpService service.RMPService,
) RoleEndpoints {
	return RoleEndpoints{
		// Roles
		CreateRole:  wrap(serviceWrapperTyped[dto.CreateRoleRequest, dto.RoleResponse](roleService.CreateRole)),
		GetRoleByID: wrapUUID(serviceWrapperByUUID[dto.RoleResponse](roleService.GetRoleByID)),
		UpdateRole:  wrap(serviceWrapperTyped[dto.UpdateRoleRequest, dto.RoleResponse](roleService.UpdateRole)),
		DeleteRole:  wrapUUIDOnly(roleService.DeleteRole),
		ListRoles:   wrapListTyped(roleService.ListRoles),

		// Modules
		CreateModule:  wrap(serviceWrapperTyped[dto.CreateModuleRequest, dto.ModuleResponse](moduleService.CreateModule)),
		GetModuleByID: wrapUUID(serviceWrapperByUUID[dto.ModuleResponse](moduleService.GetModuleByID)),
		UpdateModule:  wrap(serviceWrapperTyped[dto.UpdateModuleRequest, dto.ModuleResponse](moduleService.UpdateModule)),
		DeleteModule:  wrapUUIDOnly(moduleService.DeleteModule),
		ListModules:   wrapListTyped(moduleService.ListModules),

		// Permissions
		CreatePermission:  wrap(serviceWrapperTyped[dto.CreatePermissionRequest, dto.PermissionResponse](permissionService.CreatePermission)),
		GetPermissionByID: wrapUUID(serviceWrapperByUUID[dto.PermissionResponse](permissionService.GetPermissionByID)),
		UpdatePermission:  wrap(serviceWrapperTyped[dto.UpdatePermissionRequest, dto.PermissionResponse](permissionService.UpdatePermission)),
		DeletePermission:  wrapUUIDOnly(permissionService.DeletePermission),
		ListPermissions:   wrapListTyped(permissionService.ListPermissions),

		// RMP
		CreateRMP:  wrap(serviceWrapperTyped[dto.CreateRMPRequest, dto.RMPResponse](rmpService.CreateRMP)),
		GetRMPByID: wrapUUID(serviceWrapperByUUID[dto.RMPResponse](rmpService.GetRMPByID)),
		UpdateRMP:  wrap(serviceWrapperTyped[dto.UpdateRMPRequest, dto.RMPResponse](rmpService.UpdateRMP)),
		DeleteRMP:  wrapUUIDOnly(rmpService.DeleteRMP),
		ListRMPs:   wrapListTyped(rmpService.ListRMP),
		GetModulesAndPermissionsByRoleID: wrapUUIDList(rmpService.GetModulesAndPermissionsByRoleID),

	}
}

// Wrappers

func wrap(handler func(context.Context, interface{}) (interface{}, error)) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return handler(ctx, request)
	}
}

func wrapUUID(handler func(context.Context, uuid.UUID) (interface{}, error)) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		return handler(ctx, request.(uuid.UUID))
	}
}

func wrapUUIDOnly(handler func(context.Context, uuid.UUID) error) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		err := handler(ctx, request.(uuid.UUID))
		return map[string]string{"message": "deleted"}, err
	}
}

func wrapListTyped[T any](f func(context.Context) ([]T, error)) endpoint.Endpoint {
	return func(ctx context.Context, _ interface{}) (interface{}, error) {
		list, err := f(ctx)
		if err != nil {
			return nil, err
		}
		var result []interface{}
		for _, item := range list {
			result = append(result, item)
		}
		return result, nil
	}
}

func serviceWrapperTyped[Req any, Res any](f func(context.Context, *Req) (*Res, error)) func(context.Context, interface{}) (interface{}, error) {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		r := request.(*Req)
		return f(ctx, r)
	}
}

func serviceWrapperByUUID[Res any](f func(context.Context, uuid.UUID) (*Res, error)) func(context.Context, uuid.UUID) (interface{}, error) {
	return func(ctx context.Context, id uuid.UUID) (interface{}, error) {
		return f(ctx, id)
	}
}

func wrapUUIDList[T any](f func(context.Context, uuid.UUID) ([]T, error)) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		id := request.(uuid.UUID)
		list, err := f(ctx, id)
		if err != nil {
			return nil, err
		}
		var result []interface{}
		for _, item := range list {
			result = append(result, item)
		}
		return result, nil
	}
}


package base

import (
	"context"
	"fmt"
	"net/http"
	"time"

	commonlib "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/service"

	"errors"

	errcom "github.com/PecozQ/aimx-library/apperrors"

	"whatsdare.com/fullstack/aimx/backend/model"
)

type Endpoints struct {
	CreateTemplateEndpoint  endpoint.Endpoint
	GetTemplateByIDEndpoint endpoint.Endpoint
	UpdateTemplateEndpoint  endpoint.Endpoint
	DeleteTemplateEndpoint  endpoint.Endpoint

	CreateFormEndpoint             endpoint.Endpoint
	GetFormByTypeEndpoint          endpoint.Endpoint
	CreateFormTypeEndpoint         endpoint.Endpoint
	GetFormTypeEndpoint            endpoint.Endpoint
	UpdateFormEndpoint             endpoint.Endpoint
	GetFormFilterEndpoint          endpoint.Endpoint
	GetFormFilterBYTypeEndpoint    endpoint.Endpoint
	GetFormFilterByOrgNameEndpoint endpoint.Endpoint

	RatingDocketEndpoint    endpoint.Endpoint
	ShortlistDocketEndpoint endpoint.Endpoint
	GetCommentsByIdEndpoint endpoint.Endpoint
}

func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		CreateTemplateEndpoint:  Middleware(makeCreateTemplateEndpoint(s), commonlib.TimeoutMs),
		GetTemplateByIDEndpoint: Middleware(makeGetTemplateByTypeEndpoint(s), commonlib.TimeoutMs),
		UpdateTemplateEndpoint:  Middleware(makeUpdateTemplateEndpoint(s), commonlib.TimeoutMs),
		DeleteTemplateEndpoint:  Middleware(makeDeleteTemplateEndpoint(s), commonlib.TimeoutMs),
		// GetTemplateByIDEndpoint: Middleware(makeGetTemplateByIDEndpoint(s), common.TimeoutMs),

		CreateFormEndpoint:             Middleware(makeCreateFormEndpoint(s), commonlib.TimeoutMs),
		GetFormByTypeEndpoint:          Middleware(makeGetFormByTypeEndpoint(s), commonlib.TimeoutMs),
		CreateFormTypeEndpoint:         Middleware(makeCreateFormTypeEndpoint(s), commonlib.TimeoutMs),
		GetFormTypeEndpoint:            Middleware(makeGetFormTypeEndpoint(s), commonlib.TimeoutMs),
		UpdateFormEndpoint:             Middleware(makeUpdateFormEndpoint(s), commonlib.TimeoutMs),
		GetFormFilterEndpoint:          Middleware(makeSearchFormsEndpoint(s), commonlib.TimeoutMs),
		GetFormFilterByOrgNameEndpoint: Middleware(makeSearchFormsByOrgNameEndpoint(s), commonlib.TimeoutMs),
		// GetFormFilterBYTypeEndpoint: Middleware(makeGetFilterFieldsByTypeEndpoint(s), commonlib.TimeoutMs),

		ShortlistDocketEndpoint: Middleware(makeShortlistDocketEndpoint(s), commonlib.TimeoutMs),
		RatingDocketEndpoint:    Middleware(makeRatingDocketEndpoint(s), commonlib.TimeoutMs),
		GetCommentsByIdEndpoint: Middleware(makeGetCommentsByIdEndpoint(s), commonlib.TimeoutMs),
	}
}

func makeShortlistDocketEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dto.ShortListDTO)

		httpReq, ok := ctx.Value("HTTPRequest").(*http.Request)
		if !ok {
			return nil, errcom.NewAppError(errors.New("failed to get HTTP request from context"), http.StatusInternalServerError, "Internal error", nil)
		}
		claims, err := decodeHeaderGetClaims(httpReq)
		if err != nil {
			return nil, err
		}
		form, err := s.ShortListDocket(ctx, claims.UserID, req)
		if err != nil {
			return nil, err
		}
		return form, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}

func makeRatingDocketEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dto.RatingDTO)

		httpReq, ok := ctx.Value("HTTPRequest").(*http.Request)
		if !ok {
			return nil, service.NewAppError(errors.New("failed to get HTTP request from context"), http.StatusInternalServerError, "Internal error", nil)
		}
		claims, err := decodeHeaderGetClaims(httpReq)
		if err != nil {
			return nil, err
		}
		form, err := s.RateDocket(ctx, claims.UserID, req)
		if err != nil {
			return nil, err
		}
		return form, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}

func makeGetCommentsByIdEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(dto.ShortListDTO)
		form, err := s.GetCommentsById(ctx, req.InteractionId)
		if err != nil {
			return nil, service.NewAppError(err, http.StatusBadRequest, err.Error(), nil)
		}
		return form, nil
	}
}

// Middlewares applies both error handling and timeout middleware to an endpoint...
func Middleware(endpoint endpoint.Endpoint, timeout time.Duration) endpoint.Endpoint {
	return service.ErrorHandlingMiddleware(service.TimeoutMiddleware(5 * timeout)(endpoint))
}

func makeCreateTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entities.Template)
		template, err := s.CreateTemplate(ctx, req)
		if err != nil {
			return nil, err
		}
		return template, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}
func makeGetTemplateByTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {

		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}

		if req.ID != "" {
			// If ID is present, prioritize lookup by ID
			template, err := s.GetTemplateByType(ctx, 0, req.ID)
			if err != nil {
				return nil, err // or wrap as needed
			}
			return template, nil
		}

		if req.Type > 0 {
			// If ID is not present, use Type
			template, err := s.GetTemplateByType(ctx, req.Type, "")
			if err != nil {
				return nil, errors.New("Template Not found") // or wrap as needed
			}
			return template, nil
		}

		return nil, errors.New("either ID or Type must be provided")
	}
}

func makeUpdateTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(entities.Template)
		fmt.Println("Template ID:", req.ID.Hex())

		template, err := s.UpdateTemplate(ctx, req.ID.Hex(), req)
		if err != nil {
			return nil, err
		}
		return template, nil
	}
}

func makeDeleteTemplateEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}

		// If ID is present, prioritize lookup by ID
		res, err := s.DeleteTemplate(ctx, req.ID)
		if err != nil {
			return nil, err // or wrap as needed
		}

		return &model.Response{Message: res.Message}, nil
	}
}

func makeCreateFormEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.FormDTO)
		form, err := s.CreateForm(ctx, *req)
		if err != nil {
			fmt.Println("the err is given as")
			return nil, service.NewAppError(err, http.StatusBadRequest, err.Error(), nil)
		}
		return form, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}

func makeUpdateFormEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.UpdateFormRequest)
		fmt.Println("Form ID:", req.ID)
		fmt.Println("Form Status:", req.Status)
		strtype := req.ID

		form, err := s.UpdateForm(ctx, strtype, req.Status)
		if err != nil {
			return nil, service.NewAppError(err, http.StatusBadRequest, err.Error(), nil)
		}
		return form, nil
	}
}

func makeGetFormByTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req, ok := request.(*model.ParamRequest)
		if !ok {
			return nil, errors.New("params error")
		}
		if commonlib.IsEmpty(req) {
			return nil, errors.New("Type must be provided")
		}
		formList, err := s.GetFormByType(ctx, req.Type, req.Page, req.PageSize)
		if err != nil {
			return nil, service.NewAppError(err, http.StatusBadRequest, errcom.ErrNotFound.Error(), nil) // or wrap as needed
		}
		return formList, nil
	}
}
func makeCreateFormTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*dto.FormType)
		formtype, err := s.CreateFormType(ctx, *req)
		if err != nil {
			return model.FormTypeResponse{Error: err.Error()}, nil
		}
		return &model.FormType{ID: formtype.ID, Name: formtype.Name}, nil
		// return model.CreateUserResponse{Message: commonRepo.Create_Message, User: model.UserResponse{ID: user.ID, FirstName: user.FirstName, LastName: user.LastName, Email: user.Email, IsLocked: user.IsLocked, ProfileImage: user.ProfileImage, IsFirstLogin: user.IsFirstLogin, Role: model.UserRole{ID: role.ID, Name: role.Name}, RolePermission: user.RolePermissions}}, nil
	}
}

func makeGetFormTypeEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		formList, err := s.GetAllFormTypes(ctx)
		if err != nil {
			return nil, service.NewCustomError(errcom.ErrNotFound, err) // or wrap as needed
		}
		return formList, nil
	}
}
func makeSearchFormsEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.SearchFormsRequest)

		forms, total, err := s.GetFilteredForms(ctx, req.Type, req.SearchParam)
		if err != nil {
			return nil, service.NewAppError(err, http.StatusBadRequest, errcom.ErrNotFound.Error(), nil)
		}
		var flatForms []dto.FormDTO
		for _, f := range forms {
			if f != nil {
				flatForms = append(flatForms, *f)
			}
		}
		return &model.SearchFormsResponse{
			Forms: flatForms,
			Total: total,
		}, nil
	}
}
func makeSearchFormsByOrgNameEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(model.SearchFormsByOrganizationRequest)

		fmt.Println("Searching for form by organization name:", req.FormName)

		form, err := s.SearchFormsByOrgName(ctx, req)
		if err != nil {
			return nil, service.NewAppError(err, http.StatusBadRequest, errcom.ErrNotFound.Error(), nil)
		}

		return &model.SearchFormByNamesResponse{
			Form: form,
		}, nil
	}
}

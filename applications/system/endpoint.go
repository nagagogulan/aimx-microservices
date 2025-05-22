package base

import (
	"context"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/model"
	"whatsdare.com/fullstack/aimx/backend/service"
)

// Endpoints defines all the available endpoints.
type Endpoints struct {
	SendNotificationEndpoint    endpoint.Endpoint
	UpdateFirebaseTokenEndpoint endpoint.Endpoint
	AuditLogsEndpoint           endpoint.Endpoint
	GetAuditLogEndpoint         endpoint.Endpoint
	FindAuditLogByUserEndpoint  endpoint.Endpoint
	TestKongEndpoint            endpoint.Endpoint
}

// NewEndpoint initializes and returns an instance of Endpoints.
func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		SendNotificationEndpoint:    makeSendNotificationEndpoint(s),
		UpdateFirebaseTokenEndpoint: makeUpdateFirebaseTokenEndpoint(s),
		AuditLogsEndpoint:           makeAuditLogsEndpoint(s),
		GetAuditLogEndpoint:         makeGetAuditLogEndpoint(s),
		FindAuditLogByUserEndpoint:  makeFindAuditLogByUserEndpoint(s),
		TestKongEndpoint:            makeTestKongEndpoint(s),
	}
}

// makeSendNotificationEndpoint creates the endpoint for sending notifications.
func makeSendNotificationEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// Parse the request
		req := request.(map[string]interface{})
		userID := req["user_id"].(string)
		message := req["message"].(string)

		// Call the service method to send the notification
		err := s.SendNotification(userID, message)
		if err != nil {
			return nil, err
		}

		// Return a success message after sending the notification
		return map[string]string{"status": "success", "message": "Notification sent successfully"}, nil
	}
}

func makeUpdateFirebaseTokenEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(map[string]interface{})
		userID := req["user_id"].(string)
		token := req["firebase_token"].(string)

		err := s.UpdateFirebaseToken(userID, token)
		if err != nil {
			return nil, err
		}

		return map[string]string{"status": "success", "message": "Firebase token updated"}, nil
	}
}

func makeAuditLogsEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(map[string]interface{})
		auditLog := req["audit_log"].(*dto.AuditLogs)

		err := s.AuditLogs(ctx, auditLog)
		if err != nil {
			return nil, err
		}
		return map[string]string{"status": "success", "message": "Audit log created"}, nil
	}
}

func makeGetAuditLogEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(map[string]interface{})

		// Extract required fields from request
		role := req["role"].(string)
		orgID := req["org_id"].(string)
		if orgID == "" {
			orgID = "all"
		}
		// Extract and convert pagination parameters safely
		pageFloat, ok := req["page"].(float64)
		if !ok {
			pageFloat = 1 // default page
		}
		limitFloat, ok := req["limit"].(float64)
		if !ok {
			limitFloat = 10 // default limit
		}
		username := req["username"].(string)

		page := int(pageFloat)
		limit := int(limitFloat)

		// Call the service

		response, err := s.GetAuditLog(ctx,username,role, orgID, page, limit)
		if err != nil {
			return nil, err
		}

		return response, nil
	}
}

func makeFindAuditLogByUserEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		req := request.(*model.FindAuditByUserRequest)
		return s.FindAuditLogByUser(ctx, req.UserName, req.Page, req.Limit)
	}
}

func makeTestKongEndpoint(s service.Service) endpoint.Endpoint {
	return func(ctx context.Context, request interface{}) (interface{}, error) {
		// No request processing needed for this endpoint
		return s.TestKong(ctx)
	}
}

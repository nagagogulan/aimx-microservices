package base

import (
	"context"

	"github.com/go-kit/kit/endpoint"
	"whatsdare.com/fullstack/aimx/backend/service"
)

// Endpoints defines all the available endpoints.
type Endpoints struct {
	SendNotificationEndpoint    endpoint.Endpoint
	UpdateFirebaseTokenEndpoint endpoint.Endpoint
}

// NewEndpoint initializes and returns an instance of Endpoints.
func NewEndpoint(s service.Service) Endpoints {
	return Endpoints{
		SendNotificationEndpoint:    makeSendNotificationEndpoint(s),
		UpdateFirebaseTokenEndpoint: makeUpdateFirebaseTokenEndpoint(s),
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

package service

import (
	"fmt"
	"time"

	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/PecozQ/aimx-library/firebase"
)

type Service interface {
	SendNotification(userID, message string) error
	UpdateFirebaseToken(userID, token string) error
}

type service struct {
	repo     repository.NotificationRepo
	userRepo repository.UserCRUDService
}

func NewService(repo repository.NotificationRepo, userRepo repository.UserCRUDService,
) Service {
	return &service{
		repo:     repo,
		userRepo: userRepo,
	}
}

func (s *service) SendNotification(userID, message string) error {
	// Create a new notification entity
	notification := &entities.Notification{
		UserID:    userID,
		Message:   message,
		Token:     "user-firebase-token", // Fetch the token from the database or another service
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save the notification in the database
	err := s.repo.CreateNotification(notification)
	if err != nil {
		return err
	}

	// Convert entity to DTO
	notificationDTO := dto.Notification{
		Token:   notification.Token,
		Message: notification.Message,
	}

	// Send the push notification using Firebase
	err = firebase.SendPushNotification(notificationDTO)
	if err != nil {
		return fmt.Errorf("error sending push notification: %v", err)
	}

	return nil
}

func (s *service) UpdateFirebaseToken(userID, token string) error {
	return s.userRepo.UpdateFirebaseTokenByUserID(userID, token)
}

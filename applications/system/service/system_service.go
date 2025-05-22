package service

import (
	"context"
	"fmt"
	"math"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/PecozQ/aimx-library/firebase"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	SendNotification(userID, message string) error
	UpdateFirebaseToken(userID, token string) error
	AuditLogs(ctx context.Context, auditLog *dto.AuditLogs) error
	GetAuditLog(ctx context.Context, role string, orgID string, page int, limit int) (map[string]interface{}, error)
	FindAuditLogByUser(ctx context.Context, userID string, page, limit int) (map[string]interface{}, error)
	TestKong(ctx context.Context) (map[string]string, error)
}

type service struct {
	repo      repository.NotificationRepo
	userRepo  repository.UserCRUDService
	auditRepo repository.AuditLogsRepositoryService
}

func NewService(repo repository.NotificationRepo, userRepo repository.UserCRUDService,
	auditRepo repository.AuditLogsRepositoryService,
) Service {
	return &service{
		repo:      repo,
		userRepo:  userRepo,
		auditRepo: auditRepo,
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
		return fmt.Errorf("error sending push notification")
	}

	return nil
}

func (s *service) UpdateFirebaseToken(userID, token string) error {
	return s.userRepo.UpdateFirebaseTokenByUserID(userID, token)
}

func (s *service) AuditLogs(ctx context.Context, auditLog *dto.AuditLogs) error {
	return s.auditRepo.InsertAuditLog(ctx, auditLog)
}

func (s *service) GetAuditLog(ctx context.Context, role string, orgID string, page int, limit int) (map[string]interface{}, error) {
	// Call the repository method with pagination
	auditLogs, total, err := s.auditRepo.FilterAuditLogsByRole(ctx, role, orgID, page, limit)
	if err != nil {
		return map[string]interface{}{"data": []interface{}{}, "paging_info": model.PagingInfo{}}, errcom.ErrRecordNotFounds
	}

	// Optional: transform to flattenedData if needed, otherwise just use auditLogs
	// replace this if transformation is required

	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if len(auditLogs) == 0 {
		return map[string]interface{}{
			"data": []interface{}{},
			"paging_info": model.PagingInfo{
				TotalItems:  0,
				CurrentPage: page,
				TotalPage:   0,
				ItemPerPage: limit,
			},
		}, nil
	}
	// Return custom shape
	return map[string]interface{}{
		"data": auditLogs,
		"paging_info": model.PagingInfo{
			TotalItems:  total,
			CurrentPage: page,
			TotalPage:   totalPages,
			ItemPerPage: limit,
		},
	}, nil
}
func (s *service) FindAuditLogByUser(ctx context.Context, userName string, page, limit int) (map[string]interface{}, error) {
	logs, total, err := s.auditRepo.FindAuditlogsByUserID(ctx, userName, page, limit)
	if err != nil {
		return nil, errcom.ErrRecordNotFounds
	}
	totalPages := int(math.Ceil(float64(total) / float64(limit)))
	if len(logs) == 0 {
		return map[string]interface{}{
			"data": []interface{}{},
			"paging_info": model.PagingInfo{
				TotalItems:  0,
				CurrentPage: page,
				TotalPage:   0,
				ItemPerPage: limit,
			},
		}, nil
	}
	return map[string]interface{}{
		"data": logs,
		"paging_info": model.PagingInfo{
			TotalItems:  total,
			CurrentPage: page,
			TotalPage:   totalPages,
			ItemPerPage: limit,
		},
	}, nil
}

// TestKong is a simple endpoint to check if Kong is running
func (s *service) TestKong(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"message": "system kong up and running",
	}, nil
}

package service

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/PecozQ/aimx-library/firebase"
	"github.com/gofrs/uuid"
	"whatsdare.com/fullstack/aimx/backend/model"
)

type Service interface {
	SendNotification(userID, message string) error
	UpdateFirebaseToken(userID, token string) error
	AuditLogs(ctx context.Context, auditLog *dto.AuditLogs) error
	GetAuditLog(ctx context.Context, username string, role string, orgID string, page int, limit int) (map[string]interface{}, error)
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
	// Try to initialize Firebase if not already initialized
	if firebase.Client == nil {
		// Get Firebase credentials from environment variables
		firebaseCredentials := map[string]string{
			"FIREBASE_TYPE":                        os.Getenv("FIREBASE_TYPE"),
			"FIREBASE_PROJECT_ID":                  os.Getenv("FIREBASE_PROJECT_ID"),
			"FIREBASE_PRIVATE_KEY_ID":              os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
			"FIREBASE_PRIVATE_KEY":                 os.Getenv("FIREBASE_PRIVATE_KEY"),
			"FIREBASE_CLIENT_EMAIL":                os.Getenv("FIREBASE_CLIENT_EMAIL"),
			"FIREBASE_CLIENT_ID":                   os.Getenv("FIREBASE_CLIENT_ID"),
			"FIREBASE_AUTH_URI":                    os.Getenv("FIREBASE_AUTH_URI"),
			"FIREBASE_TOKEN_URI":                   os.Getenv("FIREBASE_TOKEN_URI"),
			"FIREBASE_AUTH_PROVIDER_X509_CERT_URL": os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
			"FIREBASE_CLIENT_X509_CERT_URL":        os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
			"FIREBASE_UNIVERSE_DOMAIN":             os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
		}

		// Initialize Firebase
		err := firebase.InitializeFirebase(firebaseCredentials)
		if err != nil {
			fmt.Printf("Warning: Failed to initialize Firebase: %v\n", err)
			fmt.Println("Push notifications will not work until Firebase is properly initialized.")
		} else {
			fmt.Println("Firebase successfully initialized")
		}
	}

	return &service{
		repo:      repo,
		userRepo:  userRepo,
		auditRepo: auditRepo,
	}
}

func (s *service) SendNotification(userID, message string) error {
	// Convert string userID to UUID
	userUUID, err := uuid.FromString(userID)
	if err != nil {
		return fmt.Errorf("invalid user ID format: %v", err)
	}

	// Get the user's Firebase token from the database with context
	ctx := context.Background()
	user, err := s.userRepo.GetUserByID(ctx, userUUID)
	if err != nil {
		return fmt.Errorf("error getting user: %v", err)
	}

	// Debug: Print user information
	fmt.Printf("User data: %+v\n", user)

	// Use the token from the user record if available
	if user == nil || user.FirebaseToken == "" {
		return fmt.Errorf("user has no valid Firebase token")
	}

	token := user.FirebaseToken

	// Debug: Print token
	fmt.Printf("Using Firebase token: %s\n", token)

	// Create a new notification entity
	notification := &entities.Notification{
		UserID:    userID,
		Message:   message,
		Token:     token,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Save the notification in the database
	err = s.repo.CreateNotification(notification)
	if err != nil {
		return err
	}

	// Check if Firebase client is initialized
	if firebase.Client == nil {
		// Get Firebase credentials from environment variables
		firebaseCredentials := map[string]string{
			"FIREBASE_TYPE":                        os.Getenv("FIREBASE_TYPE"),
			"FIREBASE_PROJECT_ID":                  os.Getenv("FIREBASE_PROJECT_ID"),
			"FIREBASE_PRIVATE_KEY_ID":              os.Getenv("FIREBASE_PRIVATE_KEY_ID"),
			"FIREBASE_PRIVATE_KEY":                 os.Getenv("FIREBASE_PRIVATE_KEY"),
			"FIREBASE_CLIENT_EMAIL":                os.Getenv("FIREBASE_CLIENT_EMAIL"),
			"FIREBASE_CLIENT_ID":                   os.Getenv("FIREBASE_CLIENT_ID"),
			"FIREBASE_AUTH_URI":                    os.Getenv("FIREBASE_AUTH_URI"),
			"FIREBASE_TOKEN_URI":                   os.Getenv("FIREBASE_TOKEN_URI"),
			"FIREBASE_AUTH_PROVIDER_X509_CERT_URL": os.Getenv("FIREBASE_AUTH_PROVIDER_X509_CERT_URL"),
			"FIREBASE_CLIENT_X509_CERT_URL":        os.Getenv("FIREBASE_CLIENT_X509_CERT_URL"),
			"FIREBASE_UNIVERSE_DOMAIN":             os.Getenv("FIREBASE_UNIVERSE_DOMAIN"),
		}

		// Debug: Print Firebase project ID
		fmt.Printf("Initializing Firebase with project ID: %s\n", firebaseCredentials["FIREBASE_PROJECT_ID"])

		// Initialize Firebase
		err = firebase.InitializeFirebase(firebaseCredentials)
		if err != nil {
			return fmt.Errorf("error initializing Firebase: %v", err)
		}
	}

	// Convert entity to DTO
	notificationDTO := dto.Notification{
		Token:   notification.Token,
		Message: notification.Message,
	}

	// Send the push notification using Firebase
	err = firebase.SendPushNotification(notificationDTO)
	if err != nil {
		if strings.Contains(err.Error(), "sender id does not match") {
			// This is a common error when the token was generated for a different Firebase project
			return fmt.Errorf("the user's FCM token was generated for a different Firebase project than the one configured on the server: %v", err)
		}
		return fmt.Errorf("error sending push notification: %v", err)
	}

	return nil
}

func (s *service) UpdateFirebaseToken(userID, token string) error {
	return s.userRepo.UpdateFirebaseTokenByUserID(userID, token)
}

func (s *service) AuditLogs(ctx context.Context, auditLog *dto.AuditLogs) error {
	return s.auditRepo.InsertAuditLog(ctx, auditLog)
}

func (s *service) GetAuditLog(ctx context.Context, username string, role string, orgID string, page int, limit int) (map[string]interface{}, error) {
	// Call the repository method with pagination
	auditLogs, total, err := s.auditRepo.FilterAuditLogsByRole(ctx, username, role, orgID, page, limit)
	if err != nil {
		return map[string]interface{}{"data": []interface{}{}, "paging_info": model.PagingInfo{}}, errcom.ErrRecordNotFounds
	}

	// Optional: transform to flattenedData if needed, otherwise just use auditLogs
	// replace this if transformation is required

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
	// Calculate total pages
	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
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

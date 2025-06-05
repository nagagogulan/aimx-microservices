package service

import (
	"context"
	"fmt"
	"net/smtp"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/repository"
	"github.com/gofrs/uuid"
)

type Service interface {
	ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string, filters map[string]interface{}, reqType string) (dto.UpdatedPaginationResponse, error)
	DeleteUser(ctx context.Context, id uuid.UUID) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ActivateUser(ctx context.Context, id uuid.UUID) error
	TestKong(ctx context.Context) (map[string]string, error)
}

type service struct {
	repo repository.UserCRUDService
}

func NewService(repo repository.UserCRUDService) Service {
	return &service{repo: repo}
}

func (s *service) ListUsers(ctx context.Context, orgID uuid.UUID, userID uuid.UUID, page, limit int, search string, filters map[string]interface{}, reqType string) (dto.UpdatedPaginationResponse, error) {
	fmt.Printf("testttt", reqType)
	if reqType == "AllOrganization" {
		return s.repo.ListUsersByCondition(ctx, nil, userID, page, limit, search, filters)
	} else {
		return s.repo.ListUsersByCondition(ctx, &orgID, userID, page, limit, search, filters)
	}

}

func (s *service) DeleteUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteUser(ctx, id)
}

func (s *service) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	// Step 1: Deactivate user
	err := s.repo.DeactivateUser(ctx, id)
	if err != nil {
		return errcom.ErrRecordNotFounds
	}

	// Step 2: Get user details to fetch email
	user, err := s.repo.GetUserByID(ctx, id)
	if err != nil {
		return errcom.ErrUserNotFound
	}
	sendEmail(user.Email)
	return nil
}

func (s *service) ActivateUser(ctx context.Context, id uuid.UUID) error {
	return s.repo.ActivateUser(ctx, id)
}

// TestKong is a simple endpoint to check if Kong is running
func (s *service) TestKong(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"message": "user kong api up and running",
	}, nil
}

func sendEmail(receiverEmail string) error {
	from := "priyadharshini.twilight@gmail.com"
	password := "rotk reak madc kwkf"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", from, password, smtpHost)
	to := []string{receiverEmail}

	message := []byte("From: SingHealth <" + from + ">\r\n" +
		"To: " + receiverEmail + "\r\n" +
		"Subject: Your Account Has Been Deactivated\r\n" +
		"Content-Type: text/html; charset=UTF-8\r\n" +
		"\r\n" +
		"<html>" +
		"<body style='font-family: Arial, sans-serif;'>" +
		"  <div style='background-color: #f8f9fa; padding: 20px;'>" +
		"    <h2 style='color: #d35400;'>⚠️ Your Account Has Been Deactivated</h2>" +
		"    <p>Dear <strong>" + receiverEmail + "</strong>,</p>" +
		"    <p>We want to inform you that your user account has been <strong>deactivated</strong> by the system administrator.</p>" +
		"    <p>This might be due to:</p>" +
		"    <ul>" +
		"      <li>Prolonged inactivity</li>" +
		"      <li>Violation of terms of service</li>" +
		"      <li>Administrative decision</li>" +
		"    </ul>" +
		"    <p>If you think this action was taken by mistake or you have any questions, please contact our support team immediately.</p>" +
		"    <p>Regards,<br><strong>SingHealth Team</strong></p>" +
		"    <p style='color: #888;'>This is an automated email. Please do not reply to this message.</p>" +
		"  </div>" +
		"</body>" +
		"</html>")

	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}

	return nil
}

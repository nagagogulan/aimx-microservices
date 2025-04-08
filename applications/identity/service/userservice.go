package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"os"
	"time"

	"database/sql"

	com "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/pquerna/otp/totp"
	"github.com/skip2/go-qrcode"
	"whatsdare.com/fullstack/aimx/backend/model"
)

var (
	otpStore = make(map[string]string) // Temporary OTP storage
)

type SMTPConfig struct {
	FromEmail string
	Password  string
	SMTPHost  string
	SMTPPort  string
}

func (s *service) SendEmailOTP(ctx context.Context, req *dto.UserAuthRequest) (*model.Response, error) {
	fmt.Println("inside function")
	if !com.ValidateEmail(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Check if user exists

	existinguser, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user: %w", err)
	}
	if existinguser != nil && existinguser.IS_MFP_Enable && existinguser.Email == req.Email {
		return &model.Response{Message: "your User name alreay exited"}, nil
	}

	// Generate OTP & Secret Key
	otp := generateOTP()

	// Store OTP & secret in DB
	if req.Email != "" {
		otpStore[req.Email] = otp
	}
	if existinguser != nil && existinguser.OTP == "" && !existinguser.IS_MFP_Enable && existinguser.Email == req.Email {
		errs := s.UserRepo.UpdateOTP(ctx, otp, existinguser.Email)
		if errs != nil {
			fmt.Println("new otp stroed error:", err)
			return nil, err
		}
	} else {
		errs := s.UserRepo.SaveOTP(ctx, req, otp)
		if errs != nil {
			fmt.Println("Failed to store OTP:", err)
			return nil, err
		}
	}
	// Send OTP via email

	if err := sendEmailOTPs(req.Email, otp); err != nil {
		fmt.Println("Failed to send OTP")
		return nil, nil
	}
	return &model.Response{Message: "OTP sent successfully"}, nil

}
func (s *service) VerifyOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.Response, error) {
	res, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("User Not Found : %w", err)
	}
	// Check OTP expiration (valid for 5 minutes)
	if time.Since(res.ExpireOTP) > 1*time.Minute {
		err := s.UserRepo.DeleteOTP(ctx, req.Email)
		if err != nil {
			return &model.Response{Message: "User not found or OTP not set."}, err
		} // Remove expired OTP
		return &model.Response{Message: "OTP expired."}, nil
	}

	// Validate OTP
	if res.OTP != req.OTP {
		return &model.Response{Message: "Invalid OTP."}, nil
	}
	errs := s.UserRepo.UpdateVerifyStatus(ctx, req.Email)
	if errs != nil {
		return &model.Response{Message: "Update verify error"}, err
	}
	// OTP is valid → Remove it from the database
	errors := s.UserRepo.DeleteOTP(ctx, req.Email)
	if errors != nil {
		return &model.Response{Message: "User not found or OTP not set."}, err
	} // Remove expired OTP
	return &model.Response{Message: "OTP Verified! User logged in successfully."}, nil
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func sendEmailOTPs(s, otp string) error {
	from := "nithiyavel402@gmail.com"
	password := "fykh tcjz emnc khed"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", from, password, smtpHost)
	to := []string{s}

	// Properly format the message
	message := []byte("From: AI Community <nithiyavel402@gmail.com>\r\n" +
		"To: " + s + "\r\n" +
		"Subject: Your OTP Code for Login Verification\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		"Your OTP is: " + otp)

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("OTP sent successfully")
	return nil
}

func (s *service) RegisterAuth(ctx context.Context, req *dto.UserAuthDetail) (*model.UserAuthResponse, error) {

	existinguser, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if existinguser != nil && existinguser.IS_MFP_Enable {
		return nil, fmt.Errorf("2FA already verified")
	}
	// Generate a new TOTP secret for the user
	secret, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "MySecureApp",
		AccountName: req.Email,
	})
	if err != nil {
		return nil, err
	}
	// Store the secret in the database
	req.Secret = secret.Secret()
	err = s.UserRepo.UpdateScreteKey(ctx, req)
	if err != nil {
		fmt.Println("Failed to store OTP:", err)
		return nil, err
	}

	// Ensure "qrcodes" directory exists
	qrDir := "./qrcodes"
	if _, err := os.Stat(qrDir); os.IsNotExist(err) {
		err = os.Mkdir(qrDir, os.ModePerm)
		if err != nil {
			return nil, err
		}
	}

	// Generate QR code image
	qrCodePath := fmt.Sprintf("%s/%s.png", qrDir, req.Email)
	err = qrcode.WriteFile(secret.URL(), qrcode.Medium, 256, qrCodePath)
	if err != nil {
		return nil, fmt.Errorf("Failed to generate QR code: %w", err)
	}

	// Return the response struct
	return &model.UserAuthResponse{
		Message: "Scan this QR code in your Authenticator App.",
		QRURL:   secret.URL(),
		QRImage: qrCodePath,
	}, nil
}

func (s *service) VerifyTOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.Response, error) {
	// Get the user's stored secret and OTP from DB
	userData, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		log.Println("Failed to fetch user details:", err)
		return &model.Response{Message: "Failed to fetch user details"}, err
	}
	if err == sql.ErrNoRows {
		return &model.Response{Message: "User not registered"}, err
	} else if err != nil {
		return &model.Response{Message: "Database error"}, err
	}

	// Validate OTP
	if !totp.Validate(req.OTP, userData.Secret) {
		return &model.Response{Message: "Invalid OTP"}, nil
	}
	err = s.UserRepo.UpdateQRVerifyStatus(ctx, req.Email)
	if err != nil {
		fmt.Println("Failed to Scan QR code verify Status:", err)
		return nil, err
	}

	return &model.Response{Message: "OTP verified successfully"}, nil
}

package service

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"net/smtp"
	"os"
	"time"

	com "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"

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

func (s *service) SendEmailOTP(ctx context.Context, req dto.UserAuthRequest) (*model.Response, error) {
	if !com.ValidateEmail(req.Email) {
		return nil, fmt.Errorf("invalid email format")
	}

	// Check if user exists

	existinguser, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if existinguser != nil {
		return &model.Response{Message: "your User name alreay exited"}, nil
	}

	// Generate OTP & Secret Key
	otp := generateOTP()

	// Store OTP & secret in DB
	if req.Email != "" {
		otpStore[req.Email] = otp
	}

	errs := s.UserRepo.SaveOTP(ctx, req, otp)
	if errs != nil {
		fmt.Println("Failed to store OTP:", err)
		return &model.Response{}, err
	}

	// Send OTP via email

	if err := sendEmailOTPs(req.Email, otp); err != nil {
		fmt.Println("Failed to send OTP")
		return &model.Response{}, nil
	}
	return &model.Response{Message: "OTP sent successfully"}, nil

}
func (s *service) VerifyOTP(ctx context.Context, req *dto.UserAuthdetail) (string, error) {
	res, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		return fmt.Sprintf("User not found or OTP not set."), err
	}

	// Check OTP expiration (valid for 5 minutes)
	if time.Since(res.CreatedAt) > 10*time.Minute {
		err := s.UserRepo.DeleteOTP(ctx, req.Email)
		if err != nil {
			return fmt.Sprintf("User not found or OTP not set."), err
		} // Remove expired OTP
		return fmt.Sprintf("OTP expired."), nil
	}

	// Validate OTP
	if res.OTP != req.OTP {
		return fmt.Sprintf("Invalid OTP."), nil
	}

	// OTP is valid → Remove it from the database
	errs := s.UserRepo.DeleteOTP(ctx, req.Email)
	if errs != nil {
		return fmt.Sprintf("User not found or OTP not set."), err
	} // Remove expired OTP
	return fmt.Sprintf("OTP Verified! User logged in successfully."), nil
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func sendEmailOTPs(s, otp string) error {
	smtpcon := SMTPConfig{
		FromEmail: os.Getenv("SMTP_EMAIL"), // Set your email in env variables
		Password:  os.Getenv("SMTP_PASS"),  // Set your app password in env variables
		SMTPHost:  "smtp.gmail.com",
		SMTPPort:  "587",
	}
	auth := smtp.PlainAuth("", smtpcon.FromEmail, smtpcon.Password, smtpcon.SMTPHost)
	to := []string{s}

	// Properly format the message
	message := []byte(
		"From: Nithya <nithiyavel402@gmail.com>\r\n" +
			"To: " + s + "\r\n" +
			"Subject: Your One-Time Password (OTP) for Verification\r\n" +
			"MIME-Version: 1.0\r\n" +
			"Content-Type: text/html; charset=UTF-8\r\n" +
			"\r\n" +
			"<html><body>" +
			"<p>Dear User,</p>" +
			"<p>Your One-Time Password (OTP) for verification is:</p>" +
			"<h2 style='color:blue;'>" + otp + "</h2>" +
			"<p>This OTP is valid for 10 minutes. Please do not share it with anyone.</p>" +
			"<p>Thank you,<br><b>Your Company Name</b></p>" +
			"</body></html>")

	// Send the email
	err := smtp.SendMail(smtpcon.SMTPHost+":"+smtpcon.SMTPPort, auth, smtpcon.FromEmail, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("OTP sent successfully")
	return nil
}

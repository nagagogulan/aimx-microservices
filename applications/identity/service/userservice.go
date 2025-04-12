package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	com "github.com/PecozQ/aimx-library/common"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/middleware"
	"github.com/pquerna/otp"
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

func (s *service) LoginWithOTP(ctx context.Context, req *dto.UserAuthRequest) (*model.Response, error) {
	fmt.Println("inside function")
	if !com.ValidateEmail(req.Email) {
		return nil, errcom.ErrInvalidEmail
	}

	// Generate OTP & Secret Key
	otp := generateOTP()

	// Check if user exists
	existingUser, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		fmt.Errorf("failed to check user: %w", err)
	}
	if existingUser != nil && existingUser.IS_MFA_Enabled {
		return &model.Response{Message: "User already exists with 2FA enabled", IS_MFA_Enabled: existingUser.IS_MFA_Enabled}, nil
	}

	// If user does not exist, save OTP as new record
	if existingUser == nil {
		err := s.UserRepo.SaveOTP(ctx, req, otp)
		if err != nil {
			fmt.Println("Failed to store OTP:", err)
			return nil, fmt.Errorf("failed to store OTP: %w", err)
		}
	} else {
		// If user exists but doesn't have an OTP and MFP is disabled, update OTP
		if existingUser != nil && !existingUser.IS_MFA_Enabled {
			err := s.UserRepo.UpdateOTP(ctx, otp, existingUser.Email)
			if err != nil {
				fmt.Println("Failed to update OTP:", err)
				return nil, fmt.Errorf("failed to update OTP: %w", err)
			}
		}
	}

	// Cache OTP temporarily (in memory map)
	if req.Email != "" {
		otpStore[req.Email] = otp
	}

	// Send OTP via email
	if err := sendEmailOTPs(req.Email, otp); err != nil {
		fmt.Println("Failed to send OTP:", err)
		return nil, fmt.Errorf("failed to send OTP")
	}

	return &model.Response{Message: "OTP sent successfully", IS_MFA_Enabled: false}, nil

}
func (s *service) VerifyOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.UserAuthResponse, error) {
	res, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		fmt.Errorf("User Not Found : %w", err)
	}
	if res != nil && req.Email != res.Email {
		return nil, NewCustomError(errcom.ErrInvalidEmail, err)
	}
	if req.OTP != res.OTP {
		return nil, NewCustomError(errcom.ErrInvalidOTP, err)
	}
	if res != nil && !res.IS_MFA_Enabled && res.Secret == "" {
		if time.Since(res.ExpireOTP) > 5*time.Minute {
			err := s.UserRepo.DeleteOTP(ctx, req.Email)
			if err != nil {
				return nil, NewCustomError(errcom.ErrNotFound, err)
			}
			return nil, NewCustomError(errcom.ErrOTPExpired, err)
		}
		errors := s.UserRepo.DeleteOTP(ctx, req.Email)
		if errors != nil {
			return nil, NewCustomError(errcom.ErrNotFound, err)
		}
		if res != nil && res.IS_MFA_Enabled {
			return nil, fmt.Errorf("2FA already verified")
		}
		// Generate a new TOTP secret for the user
		secret, err := totp.Generate(totp.GenerateOpts{
			Issuer:      "aimx-backend-app",
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
		qrCodeBytes, err := qrcode.Encode(secret.URL(), qrcode.Medium, 256)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code: %w", err)
		}

		// Encode the QR code bytes to a Base64 string
		qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCodeBytes)
		if res.OTP != "" && res.OTP == req.OTP && req.Email == res.Email {
			return &model.UserAuthResponse{Message: "OTP Verified!", QRURL: req.Secret, QRImage: qrCodeBase64}, nil
		}
		return nil, NewCustomError(errcom.ErrInvalidOTP, err)

	} else if res != nil && res.Secret != "" && !res.IS_MFA_Enabled {
		fmt.Println("Secret key already generated, regenerating QR code...")

		// Build the otpauth URI
		appName := "AI Community Portal" // replace with your app name
		userEmail := res.Email           // or use a username/identifier

		otpURL := fmt.Sprintf(
			"otpauth://totp/%s:%s?secret=%s&issuer=%s&algorithm=SHA1&digits=6&period=30",
			appName, userEmail, res.Secret, appName,
		)

		// Generate the QR code
		qrCodeBytes, err := qrcode.Encode(otpURL, qrcode.Medium, 256)
		if err != nil {
			return nil, fmt.Errorf("failed to generate QR code: %w", err)
		}

		qrCodeBase64 := base64.StdEncoding.EncodeToString(qrCodeBytes)

		return &model.UserAuthResponse{
			Message: "OTP Verified!",
			QRURL:   otpURL,
			QRImage: qrCodeBase64,
		}, nil
	}
	return nil, nil
}

func generateOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func sendEmailOTPs(s, otp string) error {
	from := "priyadharshini.twilight@gmail.com"
	password := "rotk reak madc kwkf"
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

func (s *service) VerifyTOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.Response, error) {
	// Get the user's stored secret and OTP from DB
	userData, err := s.UserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		log.Println("Failed to fetch user details:", err)
		return nil, NewCustomError(errcom.ErrNotFound, err)
	}
	qrverifystatus := userData.IS_MFA_Enabled
	isValid, err := totp.ValidateCustom(
		req.OTP,
		userData.Secret,
		time.Now().UTC(),
		totp.ValidateOpts{
			Period:    30,
			Skew:      1,
			Digits:    otp.DigitsSix,
			Algorithm: otp.AlgorithmSHA1,
		})

	if err != nil {
		fmt.Println("TOTP validation error:", err)
		return nil, NewCustomError(errcom.ErrFieldValidation, err)
	}
	if !isValid {
		fmt.Println("Invalid OTP from user")
		return nil, NewCustomError(errcom.ErrInvalidOTP, err)
	}
	if !userData.IS_MFA_Enabled {
		qrverify, err := s.UserRepo.UpdateQRVerifyStatus(ctx, req.Email)
		if err != nil {
			fmt.Println("Failed to Scan QR code verify Status:", err)
			return nil, err
		}
		qrverifystatus = qrverify.IS_MFA_Enabled
	}

	auth := middleware.TokenDetails{ // already UUID, no need to parse
		Email: req.Email,
	}
	jwtToken, err := middleware.GenerateJWT(&auth)
	if err != nil {
		return nil, err
	}
	fmt.Println("***************", jwtToken)
	return &model.Response{Message: "OTP verified successfully", JWTToken: jwtToken.AccessToken, IS_MFA_Enabled: qrverifystatus}, nil
}

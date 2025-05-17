package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"net/smtp"
	"os"
	"strings"
	"time"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/PecozQ/aimx-library/domain/dto"
	"github.com/PecozQ/aimx-library/domain/entities"
	"github.com/PecozQ/aimx-library/middleware"
	"github.com/gofrs/uuid"
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

// func init() {
// 	// Get the current working directory (from where the command is run)
// 	dir, err := os.Getwd()
// 	if err != nil {
// 		log.Fatal("Error getting current working directory:", err)
// 	}
// 	fmt.Println("Current Working Directory:", dir)

// 	// Construct the path to the .env file in the root directory
// 	envPath := filepath.Join(dir, "../.env")

// 	// Load the .env file from the correct path
// 	err = godotenv.Load(envPath)
// 	if err != nil {
// 		log.Fatal("Error loading .env file", err)
// 	}
// }

func (s *service) LoginWithOTP(ctx context.Context, req *dto.UserAuthRequest) (*model.Response, error) {

	domain := strings.Split(req.Email, "@")
	if len(domain) < 2 {
		return nil, errcom.ErrInvalidEmailFormat
	}
	org, err := s.OrgRepo.GetOrganizationByDomain(ctx, domain[1])
	if err != nil {
		return nil, errcom.ErrOrganizationNotFound
	}
	if org == nil {
		return nil, errcom.ErrOrganizationRegister
	}
	var metadata dto.OrgMetadata
	if err := json.Unmarshal(org.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal organization metadata: %w", err)
	}
	if org.CurrentUserCount >= metadata.MaxUserCount {
		return nil, errcom.ErrUserLimitReached
	}
	if org.DeletedAt != nil {
		// Check if the organization has been deactivated
		return nil, errcom.ErrOrganizationDeactivated
	}

	// Generate OTP & Secret Key
	otp := generateOTP()

	// Check if user exists
	existingUser, err := s.TempUserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		fmt.Errorf("failed to check user: %w", err)
	}
	if existingUser != nil && existingUser.IS_MFA_Enabled {
		return nil, errcom.Err2FAlreadyVerified
	}

	userDetails, err := s.UserRepo.GetUserByEmail(ctx, req.Email)
	if err != nil {
		fmt.Errorf("failed to check user: %w", err)
	}
	if userDetails.Status == entities.Deactivated {
		return &model.Response{Message: "Your account is currently deactivated. Please contact the administrator for assistance.", IS_MFA_Enabled: userDetails.IsMFAEnabled}, nil
	}
	if userDetails != nil {
		return &model.Response{Message: "User already exists with 2FA enabled", IS_MFA_Enabled: userDetails.IsMFAEnabled}, nil
	}

	// If user does not exist, save OTP as new record
	if existingUser == nil {
		err := s.TempUserRepo.SaveOTP(ctx, req, otp)
		if err != nil {
			//commonlib.LogMessage(s.logger, commonlib.Error, "Createuser", err.Error(), err, "CreateBy", req.Email)
			return nil, errcom.ErrNotFound
		}
	} else {
		// If user exists but doesn't have an OTP and MFP is disabled, update OTP
		if existingUser != nil && !existingUser.IS_MFA_Enabled {
			err := s.TempUserRepo.UpdateOTP(ctx, otp, existingUser.Email)
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
	res, err := s.TempUserRepo.GetOTPByUsername(ctx, req.Email)
	if err != nil {
		fmt.Errorf("User Not Found : %w", err)
	}

	if res != nil && req.Email != res.Email {
		return nil, errcom.ErrInvalidEmail
	}
	if req.OTP != res.OTP {
		return nil, errcom.ErrInvalidOTP
	}
	if res != nil && !res.IS_MFA_Enabled && res.Secret == "" {
		if time.Since(res.ExpireOTP) > 5*time.Minute {
			err := s.TempUserRepo.DeleteOTP(ctx, req.Email)
			if err != nil {
				return nil, errcom.ErrNotFound
			}
			return nil, errcom.ErrOTPExpired
		}
		errors := s.TempUserRepo.DeleteOTP(ctx, req.Email)
		if errors != nil {
			return nil, errcom.ErrNotFound
		}
		if res != nil && res.IS_MFA_Enabled {
			return nil, errcom.Err2FAlreadyVerified
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
		user := &entities.TempUser{
			Email:  req.Email,
			Secret: req.Secret,
		}
		err = s.TempUserRepo.UpdateScreteKey(ctx, user)
		if err != nil {
			fmt.Println("Failed to store OTP:", err)
			return nil, errcom.ErrUnabletoDelete
		}

		// Ensure "qrcodes" directory exists
		qrCodeBytes, err := qrcode.Encode(secret.URL(), qrcode.Medium, 256)
		if err != nil {
			return nil, errcom.ErrFailedToGenerateQRCode
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
			return nil, errcom.ErrFailedToGenerateQRCode
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

func sendEmailOTPs(receiverEmail, otp string) error {
	from := "priyadharshini.twilight@gmail.com"
	password := "rotk reak madc kwkf"
	smtpHost := "smtp.gmail.com"
	smtpPort := "587"
	auth := smtp.PlainAuth("", from, password, smtpHost)
	to := []string{receiverEmail}

	// Properly format the message
	// message := []byte("From: AI Community <nithiyavel402@gmail.com>\r\n" +
	// 	"To: " + s + "\r\n" +
	// 	"Subject: Your OTP Code for Login Verification\r\n" +
	// 	"Content-Type: text/plain; charset=UTF-8\r\n" +
	// 	"\r\n" +
	// 	"Your OTP is: " + otp)
	message := []byte(fmt.Sprintf("From: SingHealth <%s>\r\n"+
		"To: %s\r\n"+
		"Subject: OTP Verification\r\n"+
		"MIME-Version: 1.0\r\n"+
		"Content-Type: text/html; charset=UTF-8\r\n"+
		"\r\n"+
		`<!DOCTYPE html>
		<html>
		<head>
			<meta charset="UTF-8" />
			<title>OTP Verification</title>
			<link href="https://fonts.googleapis.com/css2?family=Open+Sans&display=swap" rel="stylesheet" />
			<style>
			body {
				font-family: 'Open Sans', sans-serif;
				background-color: #f4f4f7;
				padding: 0;
				margin: 0;
			}
			.email-container {
				max-width: 600px;
				margin: 30px auto;
				background-color: #ffffff;
				padding: 30px;
				border-radius: 10px;
				box-shadow: 0 5px 10px rgba(0, 0, 0, 0.05);
			}
			.logo {
				text-align: center;
				margin-bottom: 20px;
			}
			.otp-box {
				font-size: 28px;
				font-weight: bold;
				letter-spacing: 10px;
				background-color: #fff3ed;
				padding: 15px 25px;
				display: inline-block;
				border-radius: 8px;
				color: #F06D1A;
				margin: 20px 0;
			}
			.footer {
				font-size: 12px;
				color: #999999;
				margin-top: 30px;
				text-align: center;
			}
			@media only screen and (max-width: 620px) {
				.otp-box {
				font-size: 22px;
				letter-spacing: 6px;
				padding: 10px 20px;
				}
			}
			</style>
		</head>
		<body>
			<div class="email-container">

			<h2 style="color: #F06D1A; margin-bottom: 10px;">Verify Your Email</h2>
			<p style="font-size: 15px; color: #333;">
				Hello 👋,<br />
				Use the OTP below to verify your email address. This OTP is valid for the next <strong>10 minutes</strong>.
			</p>

			<div class="otp-box">%s</div>

			<p style="font-size: 14px; color: #555;">
				If you did not request this, please ignore this email or contact support.
			</p>

			<div class="footer">
				&copy; 2025 SingHealth. All rights reserved.<br />
				This is an automated message, please do not reply.
			</div>
			</div>
		</body>
		</html>`, from, receiverEmail, otp))

	// Send the email
	err := smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, message)
	if err != nil {
		fmt.Println("Error sending email:", err)
		return err
	}
	fmt.Println("OTP sent successfully")
	return nil
}

// Fetch JWT secrets from environment variables
func generateJWTSecrets() (string, string, error) {
	accessSecret := os.Getenv("ACCESS_SECRET")
	refreshSecret := os.Getenv("REFRESH_SECRET")

	if accessSecret == "" || refreshSecret == "" {
		return "", "", fmt.Errorf("JWT secret keys are not set in environment variables")
	}

	return accessSecret, refreshSecret, nil
}

func (s *service) VerifyTOTP(ctx context.Context, req *dto.UserAuthDetail) (*model.Response, error) {
	// Step 1: Validate the "Model" field
	// if req.Model != "" {
	// 	if req.Model != "Admin" && req.Model != "Customer" {
	// 		return nil, fmt.Errorf("invalid value for model: %s, it should be either 'Admin' or 'Customer'", req.Model)
	// 	}
	// }

	// Step 2: Get the user's OTP and validate existence
	userDataTemp, err := s.TempUserRepo.GetOTPByUsername(ctx, req.Email)
	var userData *entities.User
	if err != nil {
		// If user not found in TempUserRepo, check in UserRepo for second-time users
		userData, err = s.UserRepo.GetUserByEmail(ctx, req.Email)
		if err != nil {
			fmt.Println("Failed to fetch user details:", err)
			return nil, errcom.ErrUserNotFound
		}

		if userData != nil && userData.Status == entities.Deactivated {
			return nil, errcom.ErrUserDeactivated
		}
	}

	// Conditional user assignment based on existence in TempUserRepo or UserRepo
	var user *entities.TempUser
	if userDataTemp != nil {
		user = userDataTemp // For first-time user
	} else if userData != nil {
		user = &entities.TempUser{
			Email:          userData.Email,
			Secret:         userData.Secret,
			IS_MFA_Enabled: userData.IsMFAEnabled,
			ExpireOTP:      userData.ExpireOTP,
		}
	} else {
		return nil, errcom.ErrUserNotFound
	}

	// Step 3: Validate OTP
	isValid, err := totp.ValidateCustom(req.OTP, user.Secret, time.Now().UTC(), totp.ValidateOpts{
		Period:    30,
		Skew:      3,
		Digits:    otp.DigitsSix,
		Algorithm: otp.AlgorithmSHA1,
	})

	if err != nil || !isValid {
		return nil, errcom.ErrInvalidOTP
	}

	// Step 4: If MFA is not enabled, update QR verify status
	if !user.IS_MFA_Enabled {
		qrverify, err := s.TempUserRepo.UpdateQRVerifyStatus(ctx, req.Email)
		if err != nil {
			return nil, err
		}
		if qrverify != nil && qrverify.IS_MFA_Enabled {
			user.IS_MFA_Enabled = true
		}
	}

	// Step 5: Get organization details by email domain
	domainParts := strings.Split(req.Email, "@")
	if len(domainParts) < 2 {
		return nil, errcom.ErrInvalidEmailFormat
	}
	orgDomain := domainParts[1]

	// Fetch organization details
	org, err := s.OrgRepo.GetOrganizationByDomain(ctx, orgDomain)
	if err != nil {
		log.Println("Organization not found:", err)
		return nil, errcom.ErrOrganizationNotFound
	}

	// Step 6: Fetch SingHealthAdmin details (for role validation)
	_, err = s.OrgRepo.GetSingHealthAdminDetails(ctx)
	if err != nil {
		log.Println("Error fetching SingHealthAdmin organization details:", err)
		return nil, fmt.Errorf("no SingHealth admin organization found")
	}

	// Step 7: Fetch all roles from RoleRepo
	roles, err := s.RoleRepo.GetAllRoles(ctx)
	if err != nil {
		log.Println("Error fetching role details:", err)
		return nil, fmt.Errorf("no roles found")
	}

	// Map roles by their name for easy access (with UUID)
	roleMap := make(map[string]uuid.UUID)
	for _, role := range roles {
		roleMap[role.Name] = role.ID // Store UUID directly
	}

	// temporary function for to gget multiple admins
	organizations, err := s.OrgRepo.GetAllSingHealthAdminOrganizations(ctx)
	if err != nil {
		log.Println("Organization not found:", err)
		return nil, errcom.ErrOrganizationNotFound
	}
	// temporary function for to gget multiple admins
	emails := []string{}
	for _, org := range organizations {
		emails = append(emails, org.OrganizationEmail)
	}

	// Step 8: Determine the role based on the provided conditions
	var role uuid.UUID // Use uuid.UUID to store the role

	// actual if condtition

	// Determine role based on conditions
	// if orgDomain == org.OrganizationDomain && req.Email == org.OrganizationEmail && org.IsSingHealthAdmin {
	// 	role = roleMap["SuperAdmin"]
	// } else if orgDomain == org.OrganizationDomain && req.Email != org.OrganizationEmail && org.IsSingHealthAdmin {
	// 	role = roleMap["Collaborator"]
	// } else if orgDomain == org.OrganizationDomain && req.Email == org.OrganizationEmail && !org.IsSingHealthAdmin {
	// 	role = roleMap["Admin"]
	// } else if orgDomain == org.OrganizationDomain && req.Email != org.OrganizationEmail && !org.IsSingHealthAdmin {
	// 	role = roleMap["User"]
	// } else {
	// 	return nil, fmt.Errorf("user does not have access to login this page")
	// }

	// temporary if condition for to manage multiple role
	if orgDomain == org.OrganizationDomain && emailInList(req.Email, emails) && org.IsSingHealthAdmin {
		role = roleMap["SuperAdmin"]
	} else if orgDomain == org.OrganizationDomain && !emailInList(req.Email, emails) && org.IsSingHealthAdmin {
		role = roleMap["Collaborator"]
	} else if orgDomain == org.OrganizationDomain && req.Email == org.OrganizationEmail && !org.IsSingHealthAdmin {
		role = roleMap["Admin"]
	} else if orgDomain == org.OrganizationDomain && req.Email != org.OrganizationEmail && !org.IsSingHealthAdmin {
		role = roleMap["User"]
	} else {
		return nil, errcom.ErrUserNotAccessLogin
	}

	// Step 9: Role Validation for second-time users
	if userData != nil && userData.RoleID != role {
		return nil, errcom.ErrRoleMismatchForUser
	}

	// Step 10: If user exists, skip creation and proceed to JWT generation
	if userData != nil && userData.IsMFAEnabled {
		return s.generateJWTForExistingUser(ctx, userData, org)
	}

	// Step 11: If user doesn't exist, create a new user and assign the role
	newUser := &entities.User{
		Email:          user.Email,
		Secret:         user.Secret,
		RoleID:         role, // Use UUID value for RoleID
		OrganisationID: org.OrganizationID,
		IsMFAEnabled:   user.IS_MFA_Enabled,
		Status:         entities.Active,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Step 12: Save the new user to the database
	if err := s.UserRepo.CreateUser(ctx, newUser); err != nil {
		fmt.Println("Failed to create new user:", err)
		return nil, errcom.ErrUnbletoCreateUser
	}

	// Step 13: (Optional) Delete temp user if it was successfully moved to UserRepo
	if err := s.TempUserRepo.DeleteUser(ctx, user.ID); err != nil {
		fmt.Println("Failed to delete temp user after migration:", err)
	}

	// Step 14: Update the organization user count after successfully adding a new user
	org.CurrentUserCount++
	if err := s.OrgRepo.UpdateOrganizationByEmail(ctx, req.Email, org); err != nil {
		fmt.Println("Failed to update organization user count:", err)
		return nil, fmt.Errorf("failed to update organization user count")
	}

	// Step 15: Generate JWT for the new user
	return s.generateJWTForNewUser(ctx, newUser, org)
}

// Helper function to generate JWT for an existing user
func (s *service) generateJWTForExistingUser(ctx context.Context, userData *entities.User, org *entities.Organization) (*model.Response, error) {
	accessSecret, refreshSecret, err := generateJWTSecrets()
	if err != nil {
		fmt.Println("JWT secret keys not found:", err)
		return nil, NewCustomError(errcom.ErrNotFound, fmt.Errorf("env file not found"))
	}

	auth := middleware.TokenDetails{
		Email:          userData.Email,
		UserID:         userData.ID,
		OrganizationID: org.OrganizationID,
		AtExpires:      time.Now().Add(3 * time.Minute).Unix(),
		RtExpires:      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	jwtToken, err := middleware.GenerateJWT(&auth, []byte(accessSecret), []byte(refreshSecret))
	if err != nil {
		return nil, err
	}
	err = s.UserRepo.UpdateRefreshToken(ctx, userData.ID, jwtToken.RefreshToken)
	if err != nil {
		return nil, errcom.ErrFailedToUpdateToken
	}

	return &model.Response{Message: "OTP verified successfully", JWTToken: jwtToken.AccessToken, IS_MFA_Enabled: userData.IsMFAEnabled,
		User_Id: userData.ID, Refresh_Token: jwtToken.RefreshToken, Role_Id: userData.RoleID}, nil
}

// Helper function to generate JWT for a new user
func (s *service) generateJWTForNewUser(ctx context.Context, newUser *entities.User, org *entities.Organization) (*model.Response, error) {
	accessSecret, refreshSecret, err := generateJWTSecrets()
	if err != nil {
		fmt.Println("JWT secret keys not found")
		return nil, errcom.ErrJWTSecretNotFound
	}

	auth := middleware.TokenDetails{
		Email:          newUser.Email,
		UserID:         newUser.ID,
		OrganizationID: org.OrganizationID,
		AtExpires:      time.Now().Add(3 * time.Minute).Unix(),
		RtExpires:      time.Now().Add(7 * 24 * time.Hour).Unix(),
	}

	jwtToken, err := middleware.GenerateJWT(&auth, []byte(accessSecret), []byte(refreshSecret))
	if err != nil {
		return nil, err
	}

	err = s.UserRepo.UpdateRefreshToken(ctx, newUser.ID, jwtToken.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to update refresh token")
	}

	return &model.Response{Message: "OTP verified successfully", JWTToken: jwtToken.AccessToken, IS_MFA_Enabled: newUser.IsMFAEnabled,
		User_Id: newUser.ID, Refresh_Token: jwtToken.RefreshToken, Role_Id: newUser.RoleID}, nil
}

func (s *service) UpdateAccessToken(ctx context.Context, req *dto.RefreshAuthDetail) (*model.RefreshTokenResponse, error) {
	accessSecret, refreshSecret, err := generateJWTSecrets()
	if err != nil {
		return nil, errcom.ErrNotFound
	}

	userDetail, err := s.UserRepo.GetUserByID(ctx, req.UserId)
	if err != nil {
		return nil, errcom.ErrUserNotFound
	}

	if userDetail.RefreshToken != req.RefreshToken {
		return nil, errcom.ErrRefreshTokenInvalid
	}

	claims, err := middleware.ValidateJWT(req.RefreshToken, refreshSecret)
	if err != nil {
		return nil, errcom.ErrRefreshTokenInvalid
	}

	res, err := middleware.GenerateAccessToken(claims, []byte(accessSecret))
	if err != nil {
		return nil, errcom.ErrFailedToGrnerateToken
	}

	return &model.RefreshTokenResponse{Message: "Access token generated successfully", JWTToken: res.AccessToken}, nil
}

// temporary function for multiple admin
func emailInList(email string, emailList []string) bool {
	for _, e := range emailList {
		if e == email {
			return true
		}
	}
	return false
}

package model

import (
	"time"

	"github.com/gofrs/uuid"

)

type UserAuthRequest struct {
	Email string `json:"email"`
}
type UserAuthDetail struct {
	Email     string    `json:"email"`
	OTP       string    `json:"otp"`
	ExpireOTP time.Time `json:"expire_otp" `
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Secret    string    `json:"secret"`
	DeletedAt time.Time `json:"deleted_at"`
	Model  string `json:"model"` // Model as a string, no marshaling or conversion needed
}
type UserAuthResponse struct {
	Message string `json:"message"`
	QRURL   string `json:"qr_url"`
	QRImage string `json:"qr_image"`
}
type Response struct {
	Message        string `json:"message"`
	IS_MFA_Enabled bool   `json:"is_mfa_enabled"`
	Secret         string `json:"secret"`
	JWTToken       string `json:"jwtToken"`
	User_Id		uuid.UUID `json:"user_id"`
}

func (UserAuthDetail) TableName() string {
	return "UserAuthdetail"
}

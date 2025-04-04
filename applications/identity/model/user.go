package model

import (
	"time"
)

type UserAuthRequest struct {
	Email string `json:"email"`
}
type UserAuthdetail struct {
	Email     string     `json:"email" validate:"required,email"`
	OTP       string     `json:"otp"`
	ExpireOTP *time.Time `json:"expire_otp,omitempty" `
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	Secret    string     `json:"secret"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}
type UserAuthResponse struct {
	Message string `json:"message"`
	QRURL   string `json:"qr_url"`
	QRImage string `json:"qr_image"`
}
type Response struct {
	Message string `json:"message"`
}

func (UserAuthdetail) TableName() string {
	return "user.UserAuthdetail"
}

// func (g *UserAuthdetail) Validate() error {
// 	return validates.ValidateStruct(g)
// }

package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	errcom "github.com/PecozQ/aimx-library/apperrors"
	"github.com/go-playground/validator"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrInvalidEmail    = errors.New("invalid email format")
	ErrInvalidOTP      = errors.New("invalid OTP")
	ErrInvalidUsername = errors.New("invalid username")
	ErrOTPExpired      = errors.New("OTP expired")
	ErrUserNotFound    = errors.New("user not found")
	ErrAlreadyVerified = errors.New("2FA already verified")
)

type CustomError struct {
	ErrorType    error
	ErrorMessage error
}

// Error implements the error interface for CustomError...
func (e *CustomError) Error() string {
	return fmt.Sprintf("ErrorType: %s, Message: %s", e.ErrorType, e.ErrorMessage)
}

// NewCustomError creates a new CustomError...
func NewCustomError(errorType error, errorMessage error) *CustomError {
	return &CustomError{
		ErrorType:    errorType,
		ErrorMessage: errorMessage,
	}
}

type AppError struct {
	Err         error
	StatusCode  int
	Message     string
	FieldErrors []FieldError `json:"field_errors,omitempty"`
}
type FieldError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

type ErrorResponse struct {
	Error       string       `json:"error"`
	FieldErrors []FieldError `json:"field_errors,omitempty"`
}

func (e *AppError) Error() string {
	return e.Err.Error()
}
func NewAppError(err error, statusCode int, message string, fieldError []FieldError) *AppError {
	return &AppError{
		Err:         err,
		StatusCode:  statusCode,
		Message:     message,
		FieldErrors: fieldError,
	}
}
func EncodeError(ctx context.Context, err error, w http.ResponseWriter) {
	if appErr, ok := err.(*AppError); ok {
		w.WriteHeader(appErr.StatusCode)
		response := ErrorResponse{
			Error: appErr.Message,
		}
		if len(appErr.FieldErrors) > 0 {
			response.FieldErrors = appErr.FieldErrors
		}
		json.NewEncoder(w).Encode(response)
		return
	}
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("internal server error"))
}
func FromError(err error) *AppError {
	var pgErr *pgconn.PgError
	// Check if the error is a PostgreSQL error...
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "42703": // Undefined column
			return NewAppError(err, http.StatusBadRequest, "invalid query: column does not exist", nil)
		case "23503": // Foreign key violation
			return NewAppError(err, http.StatusConflict, "foreign key constraint fails", nil)
		case "23502": // Not null violation
			return NewAppError(err, http.StatusBadRequest, "not null constraint fails", nil)
		case "22001": // Value too long
			return NewAppError(err, http.StatusBadRequest, "value too long for column", nil)
		case "42P01": // Undefined table
			return NewAppError(err, http.StatusBadRequest, "table does not exist", nil)
		case "28P01": // Invalid password
			return NewAppError(err, http.StatusUnauthorized, "invalid credentials", nil)
		default:
			return NewAppError(err, http.StatusInternalServerError, "database error: "+pgErr.Message, nil)
		}
	} else {
		if customErr, ok := err.(*CustomError); ok {
			var errMsg string
			switch customErr.ErrorType {
			case errcom.ErrNotFound:
				errMsg = getErrorMessage(customErr.ErrorType)
				return NewAppError(err, http.StatusNotFound, errMsg, nil)
			case errcom.ErrInvalidEmailOrPassword, errcom.ErrInvalidEmail, errcom.ErrInvalidOTP, errcom.ErrOTPExpired:
				errMsg = getErrorMessage(customErr.ErrorType)
				return NewAppError(err, http.StatusBadRequest, errMsg, nil)
			case errcom.ErrFieldValidation:
				return NewAppError(err, http.StatusBadRequest, errcom.ErrValidation, FieldValidationErrors(customErr.ErrorMessage))
			default:
				return NewAppError(err, http.StatusInternalServerError, err.Error(), nil)
			}
		} else {
			if err.Error() == errcom.ErrRecordNotFound {
				return NewAppError(err, http.StatusNotFound, errcom.ErrRecordNotFound, nil)
			}
		}
		return NewAppError(err, http.StatusInternalServerError, err.Error(), nil)
	}
}

func FieldValidationErrors(err error) []FieldError {
	if ve, ok := err.(validator.ValidationErrors); ok {
		fieldErrors := make([]FieldError, len(ve))
		for i, fe := range ve {
			fieldErrors[i] = FieldError{
				Field: fe.Field(),
				Error: fe.Tag(),
			}
		}
		return fieldErrors
	}
	return []FieldError{}
}

func getErrorMessage(errorType error) string {
	switch errorType {
	case errcom.ErrNotFound:
		return errcom.ErrNotFound.Error()
	case errcom.ErrInvalidEmailOrPassword:
		return errcom.ErrInvalidEmailOrPassword.Error()
	case errcom.ErrInvalidEmail:
		return errcom.ErrInvalidEmail.Error()
	case errcom.ErrInvalidOTP:
		return errcom.ErrInvalidOTP.Error()
	case errcom.ErrOTPExpired:
		return errcom.ErrOTPExpired.Error()
	default:
		return "Unknown error"
	}
}
func ErrorEncoder(_ context.Context, err error, w http.ResponseWriter) {
	var status int

	switch {
	case strings.Contains(err.Error(), "invalid"):
		status = http.StatusBadRequest
	case strings.Contains(err.Error(), "expired"):
		status = http.StatusBadRequest
	case strings.Contains(err.Error(), "not found"):
		status = http.StatusBadRequest
	default:
		status = http.StatusInternalServerError
	}

	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{
		"error": err.Error(),
	})
}

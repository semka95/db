package web

import (
	"errors"
	"net/http"

	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

var (
	// ErrInternalServerError will throw if any the Internal Server Error happen
	ErrInternalServerError = errors.New("internal server error")
	// ErrNotFound will throw if the requested item is not exists
	ErrNotFound = errors.New("your requested item is not found")
	// ErrNoAffected will throw if no rows were affected
	ErrNoAffected = errors.New("no rows were affected")
	// ErrConflict will throw if the current action already exists
	ErrConflict = errors.New("your item already exist")
	// ErrBadParamInput will throw if the given request-body or params is not valid
	ErrBadParamInput = errors.New("given param is not valid")
	// ErrAuthenticationFailure will throw if authentication goes wrong
	ErrAuthenticationFailure = errors.New("authentication failed")
	// ErrForbidden will throw if user tries to do something that he is not
	// authorized to do
	ErrForbidden = errors.New("attempted action is not allowed")
)

// ResponseError represent the response error struct
type ResponseError struct {
	Error  string                                 `json:"error"`
	Fields validator.ValidationErrorsTranslations `json:"fields,omitempty"`
}

// GetStatusCode gets http code from error
func GetStatusCode(err error, logger *zap.Logger) int {
	if errors.Is(err, ErrAuthenticationFailure) {
		return http.StatusUnauthorized
	}
	if errors.Is(err, ErrNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, ErrConflict) {
		return http.StatusConflict
	}
	if errors.Is(err, ErrNoAffected) {
		return http.StatusNotFound
	}

	logger.Error("Server error: ", zap.Error(err))
	return http.StatusInternalServerError
}

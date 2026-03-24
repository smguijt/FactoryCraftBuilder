package apierror

import (
	"encoding/json"
	"net/http"
)

type APIError struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *APIError) Error() string { return e.Message }

func New(status int, code, message string) *APIError {
	return &APIError{Status: status, Code: code, Message: message}
}

var (
	ErrUnauthorized  = New(http.StatusUnauthorized, "unauthorized", "authentication required")
	ErrForbidden     = New(http.StatusForbidden, "forbidden", "access denied")
	ErrNotFound      = New(http.StatusNotFound, "not_found", "resource not found")
	ErrBadRequest    = func(msg string) *APIError { return New(http.StatusBadRequest, "bad_request", msg) }
	ErrConflict      = func(msg string) *APIError { return New(http.StatusConflict, "conflict", msg) }
	ErrInternal      = New(http.StatusInternalServerError, "internal_error", "internal server error")
	ErrInsufficientFunds = func(msg string) *APIError { return New(http.StatusPaymentRequired, "insufficient_funds", msg) }
)

func Write(w http.ResponseWriter, err *APIError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(err.Status)
	_ = json.NewEncoder(w).Encode(err)
}

package api

import (
	"errors"
	"fmt"
)

// APIError represents an error response from the Azure DevOps API.
type APIError struct {
	StatusCode int
	Body       string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API returned %d: %s", e.StatusCode, e.Body)
}

// IsNotFound returns true if the error is a 404 Not Found response.
func IsNotFound(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 404
}

// IsUnauthorized returns true if the error is a 401 Unauthorized response.
func IsUnauthorized(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 401
}

// IsRateLimited returns true if the error is a 429 Too Many Requests response.
func IsRateLimited(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode == 429
}

// IsServerError returns true if the error is a 5xx response.
func IsServerError(err error) bool {
	var apiErr *APIError
	return errors.As(err, &apiErr) && apiErr.StatusCode >= 500
}

// ErrorHint returns a user-friendly hint for the given error.
func ErrorHint(err error) string {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return "Press 'r' to retry"
	}
	switch {
	case apiErr.StatusCode == 401:
		return "Check your PAT or run 'az login', then press 'r'"
	case apiErr.StatusCode == 404:
		return "Resource not found \u2014 press 'R' to update repos"
	case apiErr.StatusCode == 429:
		return "Rate limited \u2014 wait a moment, then press 'r'"
	case apiErr.StatusCode >= 500:
		return "Server error \u2014 try again later, press 'r' to retry"
	default:
		return "Press 'r' to retry"
	}
}

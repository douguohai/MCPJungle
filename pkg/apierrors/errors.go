// Package apierrors defines sentinel errors shared across service and API layers.
package apierrors

import "errors"

// ErrNotFound is returned by service methods when a requested resource does not exist.
// Handlers map this to HTTP 404 via errors.Is.
var ErrNotFound = errors.New("not found")

// ErrInvalidInput is returned by service methods when user input is invalid (e.g. invalid mcp tool name).
var ErrInvalidInput = errors.New("invalid user input")

// ErrInvalidCredentials is returned when a login attempt fails (wrong username or password).
// Handlers map this to HTTP 401; the error must not reveal which of username/password was wrong.
var ErrInvalidCredentials = errors.New("invalid credentials")

// ErrUpstreamOAuthRequired indicates that the upstream server requires OAuth
// before registration can proceed.
var ErrUpstreamOAuthRequired = errors.New("upstream OAuth authorization required")

// ErrUpstreamUnavailable indicates the upstream MCP server is unhealthy or unreachable.
var ErrUpstreamUnavailable = errors.New("upstream MCP service unavailable")

// ErrPermissionDenied indicates the user is known but lacks permission for the requested resource.
var ErrPermissionDenied = errors.New("permission denied")

// ErrForbidden indicates the user is authenticated but not authorized for the
// requested action (e.g. service_admin trying to modify a server they don't manage).
var ErrForbidden = errors.New("forbidden")

// CodeUpstreamOAuthRequired is the machine-readable API error code sent when
// registration must be retried with upstream OAuth support enabled.
const CodeUpstreamOAuthRequired = "upstream_oauth_required"

// APIErrorCode is a stable, machine-readable identifier for an API error.
type APIErrorCode string

const (
	CodeUnauthenticated     APIErrorCode = "UNAUTHENTICATED"
	CodeAccountDisabled     APIErrorCode = "ACCOUNT_DISABLED"
	CodeTokenRevoked        APIErrorCode = "TOKEN_REVOKED"
	CodeTokenExpired        APIErrorCode = "TOKEN_EXPIRED"
	CodePermissionDenied    APIErrorCode = "PERMISSION_DENIED"
	CodeServiceNotOnline    APIErrorCode = "SERVICE_NOT_ONLINE"
	CodeUpstreamUnavailable APIErrorCode = "UPSTREAM_UNAVAILABLE"
	CodeUpstreamTimeout     APIErrorCode = "UPSTREAM_TIMEOUT"
	CodeToolNotFound        APIErrorCode = "TOOL_NOT_FOUND"
)

// APIError is the structured error type returned by API endpoints.
type APIError struct {
	Code    APIErrorCode `json:"code"`
	Message string       `json:"error"`
}

// NewAPIError creates an APIError with the given code and message.
func NewAPIError(code APIErrorCode, message string) *APIError {
	return &APIError{Code: code, Message: message}
}

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

// CodeUpstreamOAuthRequired is the machine-readable API error code sent when
// registration must be retried with upstream OAuth support enabled.
const CodeUpstreamOAuthRequired = "upstream_oauth_required"

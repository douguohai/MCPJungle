package apierrors

import (
	"encoding/json"
	"testing"
)

func TestAPIErrorCodeConstants(t *testing.T) {
	// Verify all error codes match their expected string values.
	tests := []struct {
		code     APIErrorCode
		expected string
	}{
		{CodeUnauthenticated, "UNAUTHENTICATED"},
		{CodeAccountDisabled, "ACCOUNT_DISABLED"},
		{CodeTokenRevoked, "TOKEN_REVOKED"},
		{CodeTokenExpired, "TOKEN_EXPIRED"},
		{CodePermissionDenied, "PERMISSION_DENIED"},
		{CodeServiceNotOnline, "SERVICE_NOT_ONLINE"},
		{CodeUpstreamUnavailable, "UPSTREAM_UNAVAILABLE"},
		{CodeUpstreamTimeout, "UPSTREAM_TIMEOUT"},
		{CodeToolNotFound, "TOOL_NOT_FOUND"},
	}
	for _, tt := range tests {
		if string(tt.code) != tt.expected {
			t.Errorf("code %v = %q, want %q", tt.code, string(tt.code), tt.expected)
		}
	}
}

func TestNewAPIError(t *testing.T) {
	err := NewAPIError(CodeUnauthenticated, "missing credentials")
	if err.Code != CodeUnauthenticated {
		t.Errorf("Code = %q, want %q", err.Code, CodeUnauthenticated)
	}
	if err.Message != "missing credentials" {
		t.Errorf("Message = %q, want %q", err.Message, "missing credentials")
	}
}

func TestAPIErrorJSON(t *testing.T) {
	err := NewAPIError(CodePermissionDenied, "access denied")
	b, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("json.Marshal failed: %v", marshalErr)
	}
	var m map[string]string
	if unmarshalErr := json.Unmarshal(b, &m); unmarshalErr != nil {
		t.Fatalf("json.Unmarshal failed: %v", unmarshalErr)
	}
	if m["code"] != "PERMISSION_DENIED" {
		t.Errorf("JSON code = %q, want %q", m["code"], "PERMISSION_DENIED")
	}
	if m["error"] != "access denied" {
		t.Errorf("JSON error = %q, want %q", m["error"], "access denied")
	}
}

func TestSentinelErrorsAreUnchanged(t *testing.T) {
	// Existing sentinel errors must still work with errors.Is.
	if ErrNotFound == nil {
		t.Error("ErrNotFound must not be nil")
	}
	if ErrInvalidInput == nil {
		t.Error("ErrInvalidInput must not be nil")
	}
	if ErrInvalidCredentials == nil {
		t.Error("ErrInvalidCredentials must not be nil")
	}
	if ErrUpstreamOAuthRequired == nil {
		t.Error("ErrUpstreamOAuthRequired must not be nil")
	}
	// New sentinels
	if ErrUpstreamUnavailable == nil {
		t.Error("ErrUpstreamUnavailable must not be nil")
	}
	if ErrPermissionDenied == nil {
		t.Error("ErrPermissionDenied must not be nil")
	}
}

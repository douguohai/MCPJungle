package types

import (
	"encoding/json"
	"testing"
)

func TestUserRole(t *testing.T) {
	t.Parallel()

	// Test UserRole constants
	if UserRoleSystemAdmin != "system_admin" {
		t.Errorf("Expected UserRoleSystemAdmin to be 'system_admin', got %s", UserRoleSystemAdmin)
	}
	if UserRoleMember != "member" {
		t.Errorf("Expected UserRoleMember to be 'member', got %s", UserRoleMember)
	}
}

func TestUser(t *testing.T) {
	t.Parallel()

	// Test struct creation
	user := User{
		Username: "testuser",
		Role:     "member",
	}

	if user.Username != "testuser" {
		t.Errorf("Expected Username to be 'testuser', got %s", user.Username)
	}
	if user.Role != "member" {
		t.Errorf("Expected Role to be 'member', got %s", user.Role)
	}
}

func TestUserZeroValues(t *testing.T) {
	t.Parallel()

	var user User

	if user.Username != "" {
		t.Errorf("Expected empty Username, got %s", user.Username)
	}
	if user.Role != "" {
		t.Errorf("Expected empty Role, got %s", user.Role)
	}
}

func TestUserJSONMarshaling(t *testing.T) {
	t.Parallel()

	user := User{
		Username: "testuser",
		Role:     "system_admin",
	}

	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal User: %v", err)
	}

	expected := `{"id":0,"username":"testuser","role":"system_admin"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestUserJSONUnmarshaling(t *testing.T) {
	t.Parallel()

	jsonData := `{"username":"testuser","role":"member"}`
	var user User

	err := json.Unmarshal([]byte(jsonData), &user)
	if err != nil {
		t.Fatalf("Failed to unmarshal User: %v", err)
	}

	if user.Username != "testuser" {
		t.Errorf("Expected Username 'testuser', got %s", user.Username)
	}
	if user.Role != "member" {
		t.Errorf("Expected Role 'member', got %s", user.Role)
	}
}

func TestCreateUserRequest(t *testing.T) {
	t.Parallel()

	// Test struct creation
	req := CreateOrUpdateUserRequest{
		Username: "newuser",
	}

	if req.Username != "newuser" {
		t.Errorf("Expected Username to be 'newuser', got %s", req.Username)
	}
}

func TestCreateUserRequestZeroValues(t *testing.T) {
	t.Parallel()

	var req CreateOrUpdateUserRequest

	if req.Username != "" {
		t.Errorf("Expected empty Username, got %s", req.Username)
	}
}

func TestCreateUserRequestJSONMarshaling(t *testing.T) {
	t.Parallel()

	req := CreateOrUpdateUserRequest{
		Username: "newuser",
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal CreateOrUpdateUserRequest: %v", err)
	}

	expected := `{"username":"newuser"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestCreateUserResponse(t *testing.T) {
	t.Parallel()

	// Test struct creation
	resp := CreateOrUpdateUserResponse{
		Username: "newuser",
		Role:     "member",
	}

	if resp.Username != "newuser" {
		t.Errorf("Expected Username to be 'newuser', got %s", resp.Username)
	}
	if resp.Role != "member" {
		t.Errorf("Expected Role to be 'member', got %s", resp.Role)
	}
}

func TestCreateUserResponseJSONMarshaling(t *testing.T) {
	t.Parallel()

	resp := CreateOrUpdateUserResponse{
		Username: "newuser",
		Role:     "system_admin",
	}

	data, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("Failed to marshal CreateOrUpdateUserResponse: %v", err)
	}

	expected := `{"username":"newuser","role":"system_admin"}`
	if string(data) != expected {
		t.Errorf("Expected JSON %s, got %s", expected, string(data))
	}
}

func TestCreateUserResponseJSONUnmarshaling(t *testing.T) {
	t.Parallel()

	jsonData := `{"username":"newuser","role":"member"}`
	var resp CreateOrUpdateUserResponse

	err := json.Unmarshal([]byte(jsonData), &resp)
	if err != nil {
		t.Fatalf("Failed to unmarshal CreateOrUpdateUserResponse: %v", err)
	}

	if resp.Username != "newuser" {
		t.Errorf("Expected Username 'newuser', got %s", resp.Username)
	}
	if resp.Role != "member" {
		t.Errorf("Expected Role 'member', got %s", resp.Role)
	}
}

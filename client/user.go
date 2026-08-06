package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// CreateUser sends a request to create a new authenticated, human user in mcpjungle
func (c *Client) CreateUser(user *types.CreateOrUpdateUserRequest) (*types.CreateOrUpdateUserResponse, error) {
	u, _ := c.constructAPIEndpoint("/users")

	body, err := json.Marshal(user)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(http.MethodPost, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, c.parseErrorResponse(resp)
	}

	var createResp types.CreateOrUpdateUserResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}

// DeleteUser sends a request to delete a user from mcpjungle
func (c *Client) DeleteUser(username string) error {
	u, _ := c.constructAPIEndpoint("/users/" + username)

	req, err := c.newRequest(http.MethodDelete, u, nil)
	if err != nil {
		return fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return c.parseErrorResponse(resp)
	}

	return nil
}

// ListUsers sends a request to list all users in mcpjungle
func (c *Client) ListUsers() ([]*types.User, error) {
	u, _ := c.constructAPIEndpoint("/users")

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var users []*types.User
	if err := json.NewDecoder(resp.Body).Decode(&users); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return users, nil
}

// Whoami sends a request to get information about the user associated with the provided access token
func (c *Client) Whoami(accessToken string) (*types.User, error) {
	u, _ := c.constructAPIEndpoint("/users/whoami")

	req, err := c.newRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var user types.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &user, nil
}

// Login authenticates with username+password and returns the web session id.
// The server sets it as an HttpOnly Set-Cookie; CLI clients extract the value
// and replay it as Authorization: Bearer on later calls. (Login lives under
// /api/dashboard/auth, outside the /v0 prefix.)
func (c *Client) Login(username, password string) (string, error) {
	u, _ := url.JoinPath(c.baseURL, "/api/dashboard/auth/login")
	payload := struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}{
		Username: username,
		Password: password,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	resp, err := c.httpClient.Post(u, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return "", fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", c.parseErrorResponse(resp)
	}
	for _, ck := range resp.Cookies() {
		if ck.Name == "mcpjungle_session" {
			return ck.Value, nil
		}
	}
	return "", fmt.Errorf("server did not set a session cookie")
}

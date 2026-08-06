package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// deviceTokenResponse is the JSON shape returned by the server for a device
// token. Defined locally to avoid coupling the client SDK to server internals.
type deviceTokenResponse struct {
	ID          uint   `json:"id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	ScopeMode   string `json:"scope_mode"`
	TokenPrefix string `json:"token_prefix"`
	ExpiresAt   string `json:"expires_at,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type createDeviceTokenResponse struct {
	RawToken string             `json:"raw_token"`
	Token    deviceTokenResponse `json:"token"`
}

// CreateDeviceToken creates a new device token for the authenticated user.
// The raw token is returned exactly once; copy it before closing the response.
func (c *Client) CreateDeviceToken(name, scopeMode string, restrictedServerNames []string) (string, *deviceTokenResponse, error) {
	u, _ := url.JoinPath(c.baseURL, "/api/dashboard/device-tokens")
	body, _ := json.Marshal(map[string]interface{}{
		"name":                    name,
		"scope_mode":              scopeMode,
		"restricted_server_names": restrictedServerNames,
	})
	req, err := http.NewRequest("POST", u, bytes.NewReader(body))
	if err != nil {
		return "", nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.accessToken)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		return "", nil, c.parseErrorResponse(resp)
	}
	var res createDeviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return "", nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return res.RawToken, &res.Token, nil
}

// ListDeviceTokens returns the caller's device tokens (admin sees all).
func (c *Client) ListDeviceTokens() ([]deviceTokenResponse, error) {
	u, _ := c.constructAPIEndpoint("/device-tokens")
	req, err := c.newRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}
	var tokens []deviceTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return tokens, nil
}

// RevokeDeviceToken revokes a device token by id.
func (c *Client) RevokeDeviceToken(id uint) error {
	u, _ := c.constructAPIEndpoint(fmt.Sprintf("/device-tokens/%d/revoke", id))
	req, err := c.newRequest("POST", u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp)
	}
	return nil
}

// DeleteDeviceToken permanently deletes a device token by id.
func (c *Client) DeleteDeviceToken(id uint) error {
	u, _ := c.constructAPIEndpoint(fmt.Sprintf("/device-tokens/%d", id))
	req, err := c.newRequest("DELETE", u, nil)
	if err != nil {
		return err
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return c.parseErrorResponse(resp)
	}
	return nil
}

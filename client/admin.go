package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/mcpjungle/mcpjungle/internal/model"
)

type InitServerResponse struct {
	AdminUsername string `json:"admin_username"`
	Message       string `json:"message"`
}

// InitServer initializes the server in enterprise mode and creates the admin
// user with the supplied credentials. The password is bcrypt-hashed server-side
// and never returned.
func (c *Client) InitServer(adminUsername, adminPassword string) (*InitServerResponse, error) {
	u, _ := url.JoinPath(c.baseURL, "/init")

	payload := struct {
		Mode          string `json:"mode"`
		AdminUsername string `json:"admin_username"`
		AdminPassword string `json:"admin_password"`
	}{
		Mode:          string(model.ModeEnterprise),
		AdminUsername: adminUsername,
		AdminPassword: adminPassword,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Post(u, "application/json", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var initResp InitServerResponse
	if err := json.NewDecoder(resp.Body).Decode(&initResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &initResp, nil
}

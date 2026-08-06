package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/mcpjungle/mcpjungle/pkg/types"
)

// CreateToolCollection sends API request to create a new Tool Collection.
func (c *Client) CreateToolCollection(collection *types.ToolCollection) (*types.CreateToolCollectionResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-collections")

	body, err := json.Marshal(collection)
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

	var createResp types.CreateToolCollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &createResp, nil
}

// DeleteToolCollection sends API request to delete a Tool Collection by name.
func (c *Client) DeleteToolCollection(name string) error {
	u, _ := c.constructAPIEndpoint("/tool-collections/" + name)

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

// ListToolCollections sends API request to list all Tool Collections.
func (c *Client) ListToolCollections() ([]types.ToolCollection, error) {
	u, _ := c.constructAPIEndpoint("/tool-collections")

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

	var collections []types.ToolCollection
	if err := json.NewDecoder(resp.Body).Decode(&collections); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return collections, nil
}

// GetToolCollection sends API request to get details of a specific Tool Collection by name.
func (c *Client) GetToolCollection(name string) (*types.GetToolCollectionResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-collections/" + name)

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

	var collection types.GetToolCollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&collection); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &collection, nil
}

// GetToolCollectionEffectiveTools sends API request to get effective tools of a specific Tool Collection by name.
func (c *Client) GetToolCollectionEffectiveTools(name string) ([]string, error) {
	u, _ := c.constructAPIEndpoint("/tool-collections/" + name + "/effective-tools")

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

	var out struct {
		Tools []string `json:"tools"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return out.Tools, nil
}

func (c *Client) UpdateToolCollection(collection *types.ToolCollection) (*types.UpdateToolCollectionResponse, error) {
	u, _ := c.constructAPIEndpoint("/tool-collections/" + collection.Name)

	body, err := json.Marshal(collection)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(http.MethodPut, u, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request to %s: %w", u, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to %s: %w", u, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseErrorResponse(resp)
	}

	var updateResp types.UpdateToolCollectionResponse
	if err := json.NewDecoder(resp.Body).Decode(&updateResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &updateResp, nil
}

// GetToolCollectionConfigs returns all Tool Collection configurations.
// It is just a user-friendly wrapper around ListToolCollections().
func (c *Client) GetToolCollectionConfigs() ([]types.ToolCollection, error) {
	return c.ListToolCollections()
}

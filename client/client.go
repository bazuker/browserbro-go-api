package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	addr   string
	client *http.Client
}

type httpMessage struct {
	Message string `json:"message"`
}

func New(serverAddress string, client *http.Client) (*Client, error) {
	if serverAddress == "" {
		return nil, errors.New("server address is required")
	}
	if !strings.HasSuffix(serverAddress, "/") {
		serverAddress += "/"
	}
	if client == nil {
		client = &http.Client{
			Timeout: 30 * time.Second,
		}
	}
	return &Client{
		addr:   serverAddress + "api/v1",
		client: client,
	}, nil
}

// Plugins fetches a list of available plugins.
func (c *Client) Plugins() ([]string, error) {
	resp, err := c.client.Get(c.addr + "/plugins")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch plugins: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected response status: %s",
			resp.Status,
		)
	}

	var plugins struct {
		Plugins []string `json:"plugins"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&plugins); err != nil {
		return nil, fmt.Errorf("failed to decode plugins: %w", err)
	}

	return plugins.Plugins, nil
}

// RunPlugin runs a plugin with the given name and parameters.
// It returns a result of the plugin execution.
func (c *Client) RunPlugin(pluginName string, params map[string]any) (map[string]any, error) {
	body, err := json.Marshal(params)
	if err != nil {
		return nil, fmt.Errorf("failed to JSON encode params: %w", err)
	}
	resp, err := c.client.Post(
		c.addr+"/plugins/"+pluginName,
		"application/json",
		bytes.NewReader(body),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to run plugin: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var msg httpMessage
		_ = json.NewDecoder(resp.Body).Decode(&msg)
		return nil, fmt.Errorf(
			"unexpected response status: %s; message: %s",
			resp.Status, msg.Message,
		)
	}

	var output map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&output); err != nil {
		return nil, fmt.Errorf("failed to decode plugin output: %w", err)
	}

	return output, nil
}

// DownloadFile downloads a file with the given ID.
func (c *Client) DownloadFile(fileID string) ([]byte, error) {
	resp, err := c.client.Get(c.addr + "/files/" + fileID)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(
			"unexpected response status: %s",
			resp.Status,
		)
	}

	return io.ReadAll(resp.Body)
}

// DeleteFile deletes a file with the given ID.
func (c *Client) DeleteFile(fileID string) error {
	req, err := http.NewRequest(http.MethodDelete, c.addr+"/files/"+fileID, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected response status: %s",
			resp.Status,
		)
	}

	return nil
}

// Healthcheck performs a health check on the server.
func (c *Client) Healthcheck() error {
	resp, err := c.client.Get(c.addr + "/health")
	if err != nil {
		return fmt.Errorf("failed to perform health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf(
			"unexpected response status: %s",
			resp.Status,
		)
	}

	return nil
}

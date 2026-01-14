package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
)

const (
	userAgent = "goasciinema/1.0.0"
)

// Client handles API communication
type Client struct {
	baseURL   string
	installID string
	client    *http.Client
}

// NewClient creates a new API client
func NewClient(baseURL, installID string) *Client {
	return &Client{
		baseURL:   baseURL,
		installID: installID,
		client:    &http.Client{},
	}
}

// UploadResponse represents the upload API response
type UploadResponse struct {
	URL     string `json:"url"`
	Message string `json:"message"`
}

// Upload uploads an asciicast file
func (c *Client) Upload(filename string) (*UploadResponse, error) {
	// Read file
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Create multipart form
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add file
	part, err := writer.CreateFormFile("asciicast", filepath.Base(filename))
	if err != nil {
		return nil, fmt.Errorf("failed to create form file: %w", err)
	}

	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	writer.Close()

	// Create request
	url := fmt.Sprintf("%s/api/asciicasts", c.baseURL)
	req, err := http.NewRequest("POST", url, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("User-Agent", c.userAgentString())
	req.Header.Set("Accept", "application/json")

	// Set basic auth
	req.SetBasicAuth("user", c.installID)

	// Send request
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var uploadResp UploadResponse
	if err := json.Unmarshal(body, &uploadResp); err != nil {
		// If response is just a URL
		uploadResp.URL = string(body)
	}

	return &uploadResp, nil
}

// AuthURL returns the URL for authentication
func (c *Client) AuthURL() string {
	return fmt.Sprintf("%s/connect/%s", c.baseURL, c.installID)
}

func (c *Client) userAgentString() string {
	return fmt.Sprintf("%s %s/%s", userAgent, runtime.GOOS, runtime.GOARCH)
}

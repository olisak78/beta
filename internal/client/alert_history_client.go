package client

import (
	"bytes"
	apperrors "developer-portal-backend/internal/errors"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// AlertHistoryClient handles communication with the external alert history service
type AlertHistoryClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewAlertHistoryClient creates a new alert history API client
func NewAlertHistoryClient(baseURL string) *AlertHistoryClient {
	return &AlertHistoryClient{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// AlertHistoryResponse represents a single alert from the external service
type AlertHistoryResponse struct {
	Fingerprint string                 `json:"fingerprint"`
	Alertname   string                 `json:"alertname"`
	Status      string                 `json:"status"`
	Severity    string                 `json:"severity"`
	Landscape   string                 `json:"landscape"`
	Region      string                 `json:"region"`
	StartsAt    string                 `json:"startsAt"`
	EndsAt      *string                `json:"endsAt"`
	Labels      map[string]interface{} `json:"labels"`
	Annotations map[string]interface{} `json:"annotations"`
	CreatedAt   string                 `json:"createdAt"`
	UpdatedAt   string                 `json:"updatedAt"`
}

// AlertHistoryPaginatedResponse represents paginated alerts from the external service
type AlertHistoryPaginatedResponse struct {
	Data       []AlertHistoryResponse `json:"data"`
	Page       int                    `json:"page"`
	PageSize   int                    `json:"pageSize"`
	TotalCount int64                  `json:"totalCount"`
	TotalPages int                    `json:"totalPages"`
}

// ProjectsResponse represents the list of available projects
type ProjectsResponse struct {
	Projects []string `json:"projects"`
}

// UpdateLabelRequest represents a label update request
type UpdateLabelRequest struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// UpdateLabelResponse represents the response after updating a label
type UpdateLabelResponse struct {
	Message     string `json:"message"`
	Project     string `json:"project"`
	Fingerprint string `json:"fingerprint"`
	Label       struct {
		Key   string `json:"key"`
		Value string `json:"value"`
	} `json:"label"`
}

// AlertFiltersResponse represents available filter values for alerts
// Uses a dynamic map to support any filter fields without hardcoding
type AlertFiltersResponse map[string][]string

// GetAvailableProjects retrieves all available projects from the alert history service
func (c *AlertHistoryClient) GetAvailableProjects() (*ProjectsResponse, error) {
	url := fmt.Sprintf("%s/api/projects", c.BaseURL)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call alert history service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alert history service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result ProjectsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAlertsByProject retrieves alerts for a specific project with filters
func (c *AlertHistoryClient) GetAlertsByProject(project string, params map[string]string) (*AlertHistoryPaginatedResponse, error) {
	// Build URL with query parameters
	baseURL := fmt.Sprintf("%s/api/alerts/%s", c.BaseURL, project)

	queryParams := url.Values{}
	for key, value := range params {
		if value != "" {
			queryParams.Add(key, value)
		}
	}

	fullURL := baseURL
	if len(queryParams) > 0 {
		fullURL = fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())
	}

	resp, err := c.HTTPClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call alert history service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alert history service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AlertHistoryPaginatedResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAlertByFingerprint retrieves a specific alert by fingerprint
func (c *AlertHistoryClient) GetAlertByFingerprint(project, fingerprint string) (*AlertHistoryResponse, error) {
	url := fmt.Sprintf("%s/api/alerts/%s/%s", c.BaseURL, project, fingerprint)

	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call alert history service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, apperrors.ErrAlertNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alert history service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AlertHistoryResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// UpdateAlertLabel updates or adds a label to an alert
func (c *AlertHistoryClient) UpdateAlertLabel(project, fingerprint string, request UpdateLabelRequest) (*UpdateLabelResponse, error) {
	url := fmt.Sprintf("%s/api/alerts/%s/%s/label", c.BaseURL, project, fingerprint)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PUT", url, bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call alert history service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, apperrors.ErrAlertNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alert history service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result UpdateLabelResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// GetAlertFilters retrieves available filter values for alerts in a specific project
func (c *AlertHistoryClient) GetAlertFilters(project string, params map[string]string) (*AlertFiltersResponse, error) {
	// Build URL with query parameters
	baseURL := fmt.Sprintf("%s/api/alerts/%s/filters", c.BaseURL, project)

	queryParams := url.Values{}
	for key, value := range params {
		if value != "" {
			queryParams.Add(key, value)
		}
	}

	fullURL := baseURL
	if len(queryParams) > 0 {
		fullURL = fmt.Sprintf("%s?%s", baseURL, queryParams.Encode())
	}

	resp, err := c.HTTPClient.Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to call alert history service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("project not found")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("alert history service returned status %d: %s", resp.StatusCode, string(body))
	}

	var result AlertFiltersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

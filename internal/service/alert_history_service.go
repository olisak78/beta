package service

import (
	"developer-portal-backend/internal/client"
	apperrors "developer-portal-backend/internal/errors"
)

// AlertHistoryService wraps the alert history API client
type AlertHistoryService struct {
	client *client.AlertHistoryClient
}

// NewAlertHistoryService creates a new alert history service
func NewAlertHistoryService(client *client.AlertHistoryClient) *AlertHistoryService {
	return &AlertHistoryService{
		client: client,
	}
}

// GetAvailableProjects retrieves all available projects
func (s *AlertHistoryService) GetAvailableProjects() (*client.ProjectsResponse, error) {
	return s.client.GetAvailableProjects()
}

// GetAlertsByProject retrieves alerts for a specific project with filters
func (s *AlertHistoryService) GetAlertsByProject(project string, filters map[string]string) (*client.AlertHistoryPaginatedResponse, error) {
	if project == "" {
		return nil, apperrors.ErrMissingProject
	}

	return s.client.GetAlertsByProject(project, filters)
}

// GetAlertByFingerprint retrieves a specific alert by fingerprint
func (s *AlertHistoryService) GetAlertByFingerprint(project, fingerprint string) (*client.AlertHistoryResponse, error) {
	if project == "" {
		return nil, apperrors.ErrMissingProject
	}
	if fingerprint == "" {
		return nil, apperrors.ErrMissingFingerprint
	}

	return s.client.GetAlertByFingerprint(project, fingerprint)
}

// UpdateAlertLabel updates or adds a label to an alert
func (s *AlertHistoryService) UpdateAlertLabel(project, fingerprint, key, value string) (*client.UpdateLabelResponse, error) {
	if project == "" {
		return nil, apperrors.ErrMissingProject
	}
	if fingerprint == "" {
		return nil, apperrors.ErrMissingFingerprint
	}
	if key == "" {
		return nil, apperrors.ErrMissingLabelKey
	}

	request := client.UpdateLabelRequest{
		Key:   key,
		Value: value,
	}

	return s.client.UpdateAlertLabel(project, fingerprint, request)
}

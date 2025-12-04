package service

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"developer-portal-backend/internal/config"
	"developer-portal-backend/internal/logger"
)

// SonarService provides methods to interact with SonarQube APIs
type SonarService struct {
	cfg        *config.Config
	httpClient *http.Client
}

// NewSonarService creates a new Sonar service
func NewSonarService(cfg *config.Config) *SonarService {
	return &SonarService{
		cfg:        cfg,
		httpClient: &http.Client{Timeout: 15 * time.Second},
	}
}

// SonarMeasure represents a single measure entry from Sonar response
type SonarMeasure struct {
	Metric    string `json:"metric"`
	Value     string `json:"value,omitempty"`
	BestValue bool   `json:"bestValue"`
}

// sonarMeasuresAPIResponse represents Sonar's measures/component API response
// Sonar returns measures under the "component" object.
type sonarMeasuresAPIResponse struct {
	Component struct {
		Measures []SonarMeasure `json:"measures"`
	} `json:"component"`
}

// sonarQualityGateAPIResponse represents Sonar's qualitygates/project_status API response
type sonarQualityGateAPIResponse struct {
	ProjectStatus struct {
		Status string `json:"status"`
	} `json:"projectStatus"`
}

// SonarCombinedResponse represents the merged response returned to the client
type SonarCombinedResponse struct {
	Measures []SonarMeasure `json:"measures"`
	Status   string         `json:"status"`
}

// GetComponentMeasures returns measures and quality gate status for a given Sonar project key.
func (s *SonarService) GetComponentMeasures(projectKey string) (*SonarCombinedResponse, error) {
	if s.cfg.SonarHost == "" || s.cfg.SonarToken == "" {
		return nil, fmt.Errorf("sonar configuration missing (SONAR_HOST or SONAR_TOKEN)")
	}
	if projectKey == "" {
		return nil, fmt.Errorf("project key (component) is required")
	}

	// Normalize base SONAR host URL
	base := s.cfg.SonarHost
	if !strings.HasPrefix(base, "http://") && !strings.HasPrefix(base, "https://") {
		base = "https://" + base
	}
	baseURL, err := url.Parse(strings.TrimRight(base, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid sonar host URL '%s': %w", base, err)
	}

	// 1) Prepare URLs for measures and quality gate
	mv := url.Values{}
	mv.Set("component", projectKey)
	mv.Set("metricKeys", "coverage,vulnerabilities,code_smells")
	measuresURL := baseURL.String() + "/api/measures/component?" + mv.Encode()

	qv := url.Values{}
	qv.Set("projectKey", projectKey)
	qgURL := baseURL.String() + "/api/qualitygates/project_status?" + qv.Encode()

	// 2) Invoke both Sonar APIs in parallel
	var measuresResp sonarMeasuresAPIResponse
	var qgResp sonarQualityGateAPIResponse
	var wg sync.WaitGroup
	var firstErr error
	var mu sync.Mutex

	wg.Add(2)
	go func() {
		defer wg.Done()
		if err := s.getJSON(measuresURL, &measuresResp); err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to fetch sonar measures: %w", err)
			}
			mu.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		if err := s.getJSON(qgURL, &qgResp); err != nil {
			mu.Lock()
			if firstErr == nil {
				firstErr = fmt.Errorf("failed to fetch sonar quality gate status: %w", err)
			}
			mu.Unlock()
		}
	}()

	wg.Wait()
	if firstErr != nil {
		return nil, firstErr
	}

	combined := &SonarCombinedResponse{
		Measures: measuresResp.Component.Measures,
		Status:   qgResp.ProjectStatus.Status,
	}
	return combined, nil
}

// getJSON performs an authenticated GET request and decodes JSON into out.
func (s *SonarService) getJSON(fullURL string, out interface{}) error {
	logger.New().Infof("Invoking Sonar API GET %s", fullURL)
	req, err := http.NewRequest(http.MethodGet, fullURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.SonarToken)
	req.Header.Set("Accept", "application/json")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sonar request failed: status=%d body=%s", resp.StatusCode, string(body))
	}

	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("failed to decode sonar response: %w", err)
	}
	return nil
}

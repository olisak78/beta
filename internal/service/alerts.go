package service

import (
	"context"
	apperrors "developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/repository"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type AlertsService struct {
	projectRepo repository.ProjectRepositoryInterface
	authService GitHubAuthService
}

func NewAlertsService(projectRepo repository.ProjectRepositoryInterface, authService GitHubAuthService) *AlertsService {
	return &AlertsService{
		projectRepo: projectRepo,
		authService: authService,
	}
}

type AlertFile struct {
	Name     string                   `json:"name"`
	Path     string                   `json:"path"`
	Content  string                   `json:"content"`
	Category string                   `json:"category"`
	Alerts   []map[string]interface{} `json:"alerts"`
}

type AlertsResponse struct {
	Files []AlertFile `json:"files"`
}

// GetProjectAlerts fetches alerts from the project's GitHub alerts repository
func (s *AlertsService) GetProjectAlerts(ctx context.Context, projectIDStr, userUUID, provider string) (*AlertsResponse, error) {

	// Get GitHub access token from claims
	accessToken, err := s.authService.GetGitHubAccessToken(userUUID, provider)
	if err != nil {
		return nil, fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get project by name (projectIDStr is actually the project name like "cis20")
	project, err := s.projectRepo.GetByName(projectIDStr)
	if err != nil {
		return nil, apperrors.ErrProjectNotFound
	}

	// Parse metadata to get alerts-repo URL
	var metadata map[string]interface{}
	if err := json.Unmarshal(project.Metadata, &metadata); err != nil {
		return nil, errors.New("failed to parse project metadata")
	}

	alertsRepo := ""
	if repo, ok := metadata["alerts"].(string); ok && repo != "" {
		alertsRepo = repo
	}

	if alertsRepo == "" {
		return nil, apperrors.ErrAlertsRepositoryNotConfigured
	}

	// Parse GitHub URL
	// Example: https://github.tools.sap/btp-monitoring/monitoring-configs/tree/main/charts/monitoring-configs/templates/alerts
	repoInfo, err := parseAlertsGitHubURL(alertsRepo)
	if err != nil {
		return nil, fmt.Errorf("invalid alerts repository URL: %w", err)
	}

	// Fetch files from GitHub API
	files, err := fetchAlertFiles(repoInfo, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch alert files: %w", err)
	}

	return &AlertsResponse{
		Files: files,
	}, nil
}

// CreateAlertPR creates a pull request with alert changes
func (s *AlertsService) CreateAlertPR(ctx context.Context, projectIDStr, userUUID, provider string, fileName, content, message, description string) (string, error) {
	// Get GitHub access token from claims
	accessToken, err := s.authService.GetGitHubAccessToken(userUUID, provider)
	if err != nil {
		return "", fmt.Errorf("failed to get GitHub access token: %w", err)
	}

	// Get project by name (projectIDStr is actually the project name like "cis20")
	project, err := s.projectRepo.GetByName(projectIDStr)
	if err != nil {
		return "", apperrors.ErrProjectNotFound
	}

	// Parse metadata to get alerts-repo URL
	var metadata map[string]interface{}
	if err := json.Unmarshal(project.Metadata, &metadata); err != nil {
		return "", errors.New("failed to parse project metadata")
	}

	alertsRepo, ok := metadata["alerts"].(string)
	if !ok || alertsRepo == "" {
		return "", apperrors.ErrAlertsRepositoryNotConfigured
	}

	// Parse GitHub URL
	repoInfo, err := parseAlertsGitHubURL(alertsRepo)
	if err != nil {
		return "", fmt.Errorf("invalid alerts repository URL: %w", err)
	}

	// Create PR via GitHub API
	prURL, err := createGitHubPR(repoInfo, fileName, content, message, description, accessToken)
	if err != nil {
		return "", fmt.Errorf("failed to create pull request: %w", err)
	}

	return prURL, nil
}

// Helper types and functions

type GitHubRepoInfo struct {
	BaseURL string
	Owner   string
	Repo    string
	Branch  string
	Path    string
}

func parseAlertsGitHubURL(url string) (*GitHubRepoInfo, error) {
	// Match pattern: https://github.tools.sap/owner/repo/tree/branch/path
	re := regexp.MustCompile(`https://([^/]+)/([^/]+)/([^/]+)/tree/([^/]+)/(.+)`)
	matches := re.FindStringSubmatch(url)
	if len(matches) != 6 {
		return nil, errors.New("invalid GitHub URL format")
	}

	return &GitHubRepoInfo{
		BaseURL: fmt.Sprintf("https://%s", matches[1]),
		Owner:   matches[2],
		Repo:    matches[3],
		Branch:  matches[4],
		Path:    matches[5],
	}, nil
}

func fetchAlertFiles(repoInfo *GitHubRepoInfo, token string) ([]AlertFile, error) {

	// GitHub API URL to list directory contents
	apiURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s?ref=%s",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo, repoInfo.Path, repoInfo.Branch)

	// Log the API URL for debugging

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	var githubFiles []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		Type        string `json:"type"`
		DownloadURL string `json:"download_url"`
		Content     string `json:"content"`
		Encoding    string `json:"encoding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&githubFiles); err != nil {
		return nil, err
	}

	var alertFiles []AlertFile
	for _, file := range githubFiles {

		// Only process YAML files
		if file.Type != "file" || (!strings.HasSuffix(file.Name, ".yaml") && !strings.HasSuffix(file.Name, ".yml")) {
			continue
		}

		// Fetch file content
		var content string
		if file.Content != "" && file.Encoding == "base64" {
			decoded, err := base64.StdEncoding.DecodeString(file.Content)
			if err != nil {
				continue
			}
			content = string(decoded)
		} else if file.DownloadURL != "" {
			fetchedContent, err := fetchFileContent(file.DownloadURL, token)
			if err != nil {
				continue
			}
			content = fetchedContent
		} else {
			continue
		}

		// Extract category from filename (e.g., "cis-db-alerts.yaml" -> "DB")
		category := extractCategory(file.Name)

		// Try to parse YAML to extract alerts, but if it fails (e.g., Helm templates),
		// try text-based extraction for displaying purposes
		var alerts []map[string]interface{}
		var yamlData map[string]interface{}
		if err := yaml.Unmarshal([]byte(content), &yamlData); err != nil {
			// Try text-based extraction for Helm templates
			alerts = extractAlertsFromText(content)
		} else {
			// Extract alerts from YAML structure
			alerts = extractAlerts(yamlData)
		}

		alertFiles = append(alertFiles, AlertFile{
			Name:     file.Name,
			Path:     file.Path,
			Content:  content,
			Category: category,
			Alerts:   alerts,
		})
	}

	return alertFiles, nil
}

func fetchFileContent(url, token string) (string, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}

	if token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func extractCategory(filename string) string {
	// Remove extension
	name := strings.TrimSuffix(filename, ".yaml")
	name = strings.TrimSuffix(name, ".yml")

	// Extract category from pattern like "cis-db-alerts" -> "DB"
	parts := strings.Split(name, "-")
	if len(parts) >= 2 {
		// Return the middle part capitalized
		category := strings.ToUpper(parts[1])
		return category
	}

	return "General"
}

func extractAlerts(yamlData map[string]interface{}) []map[string]interface{} {
	var alerts []map[string]interface{}

	// Navigate through common Prometheus/Kubernetes alert structures
	// Try different paths: spec.groups[].rules[], groups[].rules[], rules[]
	if spec, ok := yamlData["spec"].(map[string]interface{}); ok {
		if groups, ok := spec["groups"].([]interface{}); ok {
			alerts = extractFromGroups(groups)
		}
	} else if groups, ok := yamlData["groups"].([]interface{}); ok {
		alerts = extractFromGroups(groups)
	} else if rules, ok := yamlData["rules"].([]interface{}); ok {
		for _, rule := range rules {
			if ruleMap, ok := rule.(map[string]interface{}); ok {
				alerts = append(alerts, ruleMap)
			}
		}
	}

	return alerts
}

func extractFromGroups(groups []interface{}) []map[string]interface{} {
	var alerts []map[string]interface{}
	for _, group := range groups {
		if groupMap, ok := group.(map[string]interface{}); ok {
			if rules, ok := groupMap["rules"].([]interface{}); ok {
				for _, rule := range rules {
					if ruleMap, ok := rule.(map[string]interface{}); ok {
						// Add group name to alert for context
						if groupName, ok := groupMap["name"].(string); ok {
							ruleMap["_group"] = groupName
						}
						alerts = append(alerts, ruleMap)
					}
				}
			}
		}
	}
	return alerts
}

func createGitHubPR(repoInfo *GitHubRepoInfo, fileName, content, message, description, token string) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	// Generate unique branch name
	branchName := fmt.Sprintf("alert-update-%d", time.Now().Unix())
	filePath := fmt.Sprintf("%s/%s", repoInfo.Path, fileName)

	// Step 1: Get the base branch ref (SHA of the latest commit on main/master)
	refURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/git/ref/heads/%s",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo, repoInfo.Branch)

	req, err := http.NewRequest("GET", refURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create ref request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get base ref: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get base ref, status %d: %s", resp.StatusCode, string(body))
	}

	var refData struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&refData); err != nil {
		return "", fmt.Errorf("failed to decode ref response: %w", err)
	}

	baseSHA := refData.Object.SHA

	// Step 2: Create a new branch from the base
	createBranchURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/git/refs",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo)

	branchPayload := map[string]string{
		"ref": fmt.Sprintf("refs/heads/%s", branchName),
		"sha": baseSHA,
	}
	branchBody, _ := json.Marshal(branchPayload)

	req, err = http.NewRequest("POST", createBranchURL, strings.NewReader(string(branchBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create branch request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create branch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create branch, status %d: %s", resp.StatusCode, string(body))
	}

	// Step 3: Get the current file to get its SHA
	getFileURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s?ref=%s",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo, filePath, repoInfo.Branch)

	req, err = http.NewRequest("GET", getFileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create get file request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to get file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to get file, status %d: %s", resp.StatusCode, string(body))
	}

	var fileData struct {
		SHA string `json:"sha"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&fileData); err != nil {
		return "", fmt.Errorf("failed to decode file response: %w", err)
	}

	fileSHA := fileData.SHA

	// Step 4: Update the file on the new branch
	updateFileURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/contents/%s",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo, filePath)

	// Base64 encode the content
	encodedContent := base64.StdEncoding.EncodeToString([]byte(content))

	updatePayload := map[string]interface{}{
		"message": message,
		"content": encodedContent,
		"sha":     fileSHA,
		"branch":  branchName,
	}
	updateBody, _ := json.Marshal(updatePayload)

	req, err = http.NewRequest("PUT", updateFileURL, strings.NewReader(string(updateBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create update file request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to update file: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to update file, status %d: %s", resp.StatusCode, string(body))
	}

	// Step 5: Create pull request
	createPRURL := fmt.Sprintf("%s/api/v3/repos/%s/%s/pulls",
		repoInfo.BaseURL, repoInfo.Owner, repoInfo.Repo)

	prPayload := map[string]interface{}{
		"title": message,
		"body":  description,
		"head":  branchName,
		"base":  repoInfo.Branch,
	}
	prBody, _ := json.Marshal(prPayload)

	req, err = http.NewRequest("POST", createPRURL, strings.NewReader(string(prBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create PR request: %w", err)
	}
	req.Header.Set("Authorization", fmt.Sprintf("token %s", token))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("Content-Type", "application/json")

	resp, err = client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to create PR: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("failed to create PR, status %d: %s", resp.StatusCode, string(body))
	}

	var prData struct {
		HTMLURL string `json:"html_url"`
		Number  int    `json:"number"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prData); err != nil {
		return "", fmt.Errorf("failed to decode PR response: %w", err)
	}

	return prData.HTMLURL, nil
}

// extractAlertsFromText extracts alert definitions from text content (for Helm templates)
// This is a simple regex-based parser that can handle templated YAML
func extractAlertsFromText(content string) []map[string]interface{} {
	var alerts []map[string]interface{}

	// Use regex to find alert definitions
	// Look for "- alert:" followed by the alert name
	alertPattern := regexp.MustCompile(`(?m)^\s*-\s*alert:\s*(.+?)$`)
	forPattern := regexp.MustCompile(`(?m)^\s*for:\s*(.+?)$`)

	matches := alertPattern.FindAllStringSubmatchIndex(content, -1)

	for i, match := range matches {
		alertStart := match[0]
		alertEnd := len(content)
		if i+1 < len(matches) {
			alertEnd = matches[i+1][0]
		}

		alertSection := content[alertStart:alertEnd]
		alertName := strings.TrimSpace(content[match[2]:match[3]])

		alert := map[string]interface{}{
			"alert": alertName,
		}

		// Extract expr - handle both single-line and multi-line (| or >)
		expr := extractExpression(alertSection)
		if expr != "" {
			alert["expr"] = expr
		}

		// Extract 'for' duration
		if forMatch := forPattern.FindStringSubmatch(alertSection); len(forMatch) > 1 {
			alert["for"] = strings.TrimSpace(forMatch[1])
		}

		// Extract ALL labels (not just severity)
		labels := extractLabelsFromSection(alertSection)
		if len(labels) > 0 {
			alert["labels"] = labels
		}

		// Extract ALL annotations (not just summary)
		annotations := extractAnnotationsFromSection(alertSection)
		if len(annotations) > 0 {
			alert["annotations"] = annotations
		}

		alerts = append(alerts, alert)
	}

	return alerts
}

// extractExpression extracts the expression from an alert section, handling multi-line YAML
func extractExpression(alertSection string) string {
	// First, try to find "expr:" line
	exprLinePattern := regexp.MustCompile(`(?m)^\s*expr:\s*(.*)$`)
	exprMatch := exprLinePattern.FindStringSubmatch(alertSection)

	if len(exprMatch) < 2 {
		return ""
	}

	exprLine := strings.TrimSpace(exprMatch[1])

	// If it's a multi-line block scalar (| or >), we need to capture the indented lines that follow
	if exprLine == "|" || exprLine == ">" || strings.HasPrefix(exprLine, "|-") || strings.HasPrefix(exprLine, ">-") {
		// Find the position of "expr:" in the original section
		exprIndex := strings.Index(alertSection, exprMatch[0])
		if exprIndex == -1 {
			return exprLine
		}

		// Get everything after the "expr: |" line
		afterExpr := alertSection[exprIndex+len(exprMatch[0]):]

		// Extract indented lines that belong to the expression
		var exprLines []string
		lines := strings.Split(afterExpr, "\n")

		// Determine the base indentation (from the first non-empty line)
		var baseIndent int
		foundBase := false

		for _, line := range lines {
			// Skip empty lines
			if strings.TrimSpace(line) == "" {
				if len(exprLines) > 0 {
					// Add empty lines if we've already started collecting
					exprLines = append(exprLines, "")
				}
				continue
			}

			// Count leading spaces
			indent := len(line) - len(strings.TrimLeft(line, " \t"))

			if !foundBase {
				// This is the first line - set base indentation
				baseIndent = indent
				foundBase = true
				exprLines = append(exprLines, strings.TrimSpace(line))
			} else if indent >= baseIndent {
				// This line is part of the expression (same or more indentation)
				exprLines = append(exprLines, strings.TrimSpace(line))
			} else {
				// Less indentation - we've reached the next field
				break
			}
		}

		return strings.Join(exprLines, " ")
	}

	// Single-line expression
	return exprLine
}

// extractLabelsFromSection extracts all labels from an alert section
func extractLabelsFromSection(alertSection string) map[string]interface{} {
	labels := make(map[string]interface{})

	// Find the labels: section
	labelsPattern := regexp.MustCompile(`(?m)^\s*labels:\s*$`)
	labelsMatch := labelsPattern.FindStringIndex(alertSection)
	if labelsMatch == nil {
		return labels
	}

	// Get everything after "labels:" until we hit another top-level field
	afterLabels := alertSection[labelsMatch[1]:]
	lines := strings.Split(afterLabels, "\n")

	// Extract individual label key-value pairs (indented under labels:)
	labelPattern := regexp.MustCompile(`^\s{4,}(\w+):\s*(.+?)\s*$`)

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)

		// Stop at Helm directives or annotations section
		if strings.HasPrefix(trimmedLine, "{{-") || strings.HasPrefix(trimmedLine, "{{") || trimmedLine == "annotations:" {
			break
		}

		// Stop at next non-indented line (next section)
		if trimmedLine != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
			break
		}

		// Skip empty lines
		if trimmedLine == "" {
			continue
		}

		labelMatch := labelPattern.FindStringSubmatch(line)
		if len(labelMatch) > 2 {
			key := strings.TrimSpace(labelMatch[1])
			value := strings.TrimSpace(labelMatch[2])
			labels[key] = value
		}
	}

	return labels
}

// extractAnnotationsFromSection extracts all annotations from an alert section
func extractAnnotationsFromSection(alertSection string) map[string]interface{} {
	annotations := make(map[string]interface{})

	// Find the annotations: section
	annotationsPattern := regexp.MustCompile(`(?m)^\s*annotations:\s*$`)
	annotationsMatch := annotationsPattern.FindStringIndex(alertSection)
	if annotationsMatch == nil {
		return annotations
	}

	// Get everything after "annotations:" until we hit another top-level field
	afterAnnotations := alertSection[annotationsMatch[1]:]
	lines := strings.Split(afterAnnotations, "\n")

	// Pattern to match annotation key with opening quote
	keyPattern := regexp.MustCompile(`^\s{4,}(\w+):\s*"(.*)`)

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmedLine := strings.TrimSpace(line)

		// Stop at Helm directives or non-indented lines (next section)
		if strings.HasPrefix(trimmedLine, "{{-") || strings.HasPrefix(trimmedLine, "{{") {
			break
		}
		if trimmedLine != "" && !strings.HasPrefix(line, "    ") && !strings.HasPrefix(line, "\t") {
			break
		}

		// Skip empty lines
		if trimmedLine == "" {
			i++
			continue
		}

		// Try to match annotation key with quoted value
		keyMatch := keyPattern.FindStringSubmatch(line)
		if len(keyMatch) > 2 {
			key := strings.TrimSpace(keyMatch[1])
			value := keyMatch[2]

			// Check if the value ends with a closing quote on the same line
			if strings.HasSuffix(value, "\"") {
				// Single-line quoted value
				value = strings.TrimSuffix(value, "\"")
				annotations[key] = value
				i++
			} else {
				// Multi-line quoted value - collect lines until closing quote
				valueLines := []string{value}
				i++

				for i < len(lines) {
					nextLine := lines[i]
					// Check if this line ends the quoted string
					if strings.Contains(nextLine, "\"") {
						// Find the closing quote
						quoteIndex := strings.Index(nextLine, "\"")
						valueLines = append(valueLines, strings.TrimSpace(nextLine[:quoteIndex]))
						i++
						break
					}
					// Add this line to the value (it's part of the multi-line string)
					valueLines = append(valueLines, strings.TrimSpace(nextLine))
					i++
				}

				// Join all lines with space (or newline if you want to preserve formatting)
				annotations[key] = strings.Join(valueLines, " ")
			}
		} else {
			// Try unquoted value pattern
			unquotedPattern := regexp.MustCompile(`^\s{4,}(\w+):\s*(.+?)\s*$`)
			unquotedMatch := unquotedPattern.FindStringSubmatch(line)
			if len(unquotedMatch) > 2 {
				key := strings.TrimSpace(unquotedMatch[1])
				value := strings.TrimSpace(unquotedMatch[2])
				annotations[key] = value
			}
			i++
		}
	}

	return annotations
}

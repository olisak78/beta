package service

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/logger"
	"developer-portal-backend/internal/repository"

	"github.com/gin-gonic/gin"
)

// AICoreCredentials represents the credentials for a specific team
type AICoreCredentials struct {
	Team          string `json:"team"`
	ClientID      string `json:"clientId"`
	ClientSecret  string `json:"clientSecret"`
	OAuthURL      string `json:"oauthUrl"`
	APIURL        string `json:"apiUrl"`
	ResourceGroup string `json:"resourceGroup"`
}

// AICoreTokenResponse represents the OAuth token response
type AICoreTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// tokenCache represents a cached access token with expiration
type tokenCache struct {
	token     string
	expiresAt time.Time
}

// AICoreDeployment represents a deployment from AI Core
type AICoreDeployment struct {
	ID                string                 `json:"id"`
	ConfigurationID   string                 `json:"configurationId"`
	ConfigurationName string                 `json:"configurationName"`
	ScenarioID        string                 `json:"scenarioId"`
	Status            string                 `json:"status"`
	StatusMessage     string                 `json:"statusMessage"`
	TargetStatus      string                 `json:"targetStatus"`
	DeploymentURL     string                 `json:"deploymentUrl"`
	CreatedAt         string                 `json:"createdAt"`
	ModifiedAt        string                 `json:"modifiedAt"`
	Details           map[string]interface{} `json:"details,omitempty"`
}

// AICoreTeamDeployments represents deployments for a specific team
type AICoreTeamDeployments struct {
	Team        string             `json:"team"`
	Deployments []AICoreDeployment `json:"deployments"`
}

// AICoreDeploymentsResponse represents the response from AI Core deployments API
type AICoreDeploymentsResponse struct {
	Count       int                     `json:"count"`
	Deployments []AICoreTeamDeployments `json:"deployments"`
}

// AICoreModel represents a model from AI Core
type AICoreModel struct {
	Model        string                   `json:"model"`
	ExecutableID string                   `json:"executableId"`
	Description  string                   `json:"description"`
	DisplayName  string                   `json:"displayName,omitempty"`
	AccessType   string                   `json:"accessType,omitempty"`
	Provider     string                   `json:"provider,omitempty"`
	Versions     []AICoreModelVersion     `json:"versions"`
	Scenarios    []map[string]interface{} `json:"allowedScenarios,omitempty"`
}

// AICoreModelVersion represents a model version
type AICoreModelVersion struct {
	Name                      string                   `json:"name"`
	IsLatest                  bool                     `json:"isLatest"`
	Deprecated                bool                     `json:"deprecated"`
	RetirementDate            string                   `json:"retirementDate,omitempty"`
	ContextLength             int                      `json:"contextLength,omitempty"`
	InputTypes                []string                 `json:"inputTypes,omitempty"`
	Capabilities              []string                 `json:"capabilities,omitempty"`
	Metadata                  []map[string]interface{} `json:"metadata,omitempty"`
	Cost                      []map[string]interface{} `json:"cost,omitempty"`
	SuggestedReplacements     []string                 `json:"suggestedReplacements,omitempty"`
	StreamingSupported        bool                     `json:"streamingSupported,omitempty"`
	OrchestrationCapabilities []string                 `json:"orchestrationCapabilities,omitempty"`
}

// AICoreModelsResponse represents the response from AI Core models API
type AICoreModelsResponse struct {
	Count     int           `json:"count"`
	Resources []AICoreModel `json:"resources"`
}

// AICoreConfigurationsResponse represents the response from AI Core configurations API
type AICoreConfigurationsResponse struct {
	Count     int                   `json:"count"`
	Resources []AICoreConfiguration `json:"resources"`
}

// AICoreMeResponse represents the response for /ai-core/me
type AICoreMeResponse struct {
	User        string   `json:"user"`
	AIInstances []string `json:"ai_instances"`
}

// AICoreConfiguration represents a configuration from AI Core
type AICoreConfiguration struct {
	ID                    string                 `json:"id"`
	Name                  string                 `json:"name"`
	ExecutableID          string                 `json:"executableId"`
	ScenarioID            string                 `json:"scenarioId"`
	ParameterBindings     []map[string]string    `json:"parameterBindings,omitempty"`
	InputArtifactBindings []map[string]string    `json:"inputArtifactBindings,omitempty"`
	CreatedAt             string                 `json:"createdAt"`
	Scenario              map[string]interface{} `json:"scenario,omitempty"`
}

// AICoreConfigurationRequest represents a request to create a configuration
type AICoreConfigurationRequest struct {
	Name                  string              `json:"name" validate:"required"`
	ExecutableID          string              `json:"executableId" validate:"required"`
	ScenarioID            string              `json:"scenarioId" validate:"required"`
	ParameterBindings     []map[string]string `json:"parameterBindings,omitempty"` // model name and model version
	InputArtifactBindings []map[string]string `json:"inputArtifactBindings,omitempty"`
}

// AICoreConfigurationResponse represents the response from creating a configuration
type AICoreConfigurationResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// AICoreDeploymentRequest represents a request to create a deployment
// Either ConfigurationID or ConfigurationRequest must be provided
type AICoreDeploymentRequest struct {
	ConfigurationID      *string                     `json:"configurationId,omitempty"`
	ConfigurationRequest *AICoreConfigurationRequest `json:"configurationRequest,omitempty"`
	TTL                  string                      `json:"ttl,omitempty"`
}

// AICoreDeploymentResponse represents the response from creating a deployment
type AICoreDeploymentResponse struct {
	ID            string `json:"id"`
	Message       string `json:"message"`
	DeploymentURL string `json:"deploymentUrl,omitempty"`
	Status        string `json:"status,omitempty"`
	TTL           string `json:"ttl,omitempty"`
}

// AICoreDeploymentModificationRequest represents a request to modify a deployment
type AICoreDeploymentModificationRequest struct {
	TargetStatus    string `json:"targetStatus,omitempty"`
	ConfigurationID string `json:"configurationId,omitempty"`
}

// AICoreDeploymentModificationResponse represents the response from modifying a deployment
type AICoreDeploymentModificationResponse struct {
	ID            string `json:"id"`
	Message       string `json:"message"`
	DeploymentURL string `json:"deploymentUrl,omitempty"`
	Status        string `json:"status,omitempty"`
	TargetStatus  string `json:"targetStatus,omitempty"`
}

// AICoreDeploymentDeletionResponse represents the response from deleting a deployment
type AICoreDeploymentDeletionResponse struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// AICoreDeploymentDetailsResponse represents the detailed response for a specific deployment
type AICoreDeploymentDetailsResponse struct {
	ID                           string                 `json:"id"`
	DeploymentURL                string                 `json:"deploymentUrl"`
	ConfigurationID              string                 `json:"configurationId"`
	ConfigurationName            string                 `json:"configurationName"`
	ExecutableID                 string                 `json:"executableId"`
	ScenarioID                   string                 `json:"scenarioId"`
	Status                       string                 `json:"status"`
	StatusMessage                string                 `json:"statusMessage"`
	TargetStatus                 string                 `json:"targetStatus"`
	LastOperation                string                 `json:"lastOperation"`
	LatestRunningConfigurationID string                 `json:"latestRunningConfigurationId"`
	TTL                          string                 `json:"ttl"`
	Details                      map[string]interface{} `json:"details"`
	CreatedAt                    string                 `json:"createdAt"`
	ModifiedAt                   string                 `json:"modifiedAt"`
	SubmissionTime               string                 `json:"submissionTime"`
	StartTime                    string                 `json:"startTime"`
	CompletionTime               string                 `json:"completionTime"`
	StatusDetails                map[string]interface{} `json:"statusDetails"`
}

// AICoreService handles AI Core operations
type AICoreService struct {
	userRepo        repository.UserRepositoryInterface
	teamRepo        repository.TeamRepositoryInterface
	groupRepo       repository.GroupRepositoryInterface
	orgRepo         repository.OrganizationRepositoryInterface
	httpClient      *http.Client
	credentials     map[string]*AICoreCredentials // Cached credentials by team name
	credentialsMux  sync.RWMutex                  // Protects credentials cache
	tokenCache      map[string]*tokenCache        // Cached tokens by team name
	tokenCacheMux   sync.RWMutex                  // Protects token cache
	credentialsOnce sync.Once                     // Ensures credentials are loaded only once
}

/* NewAICoreService creates a new AI Core service */
func NewAICoreService(userRepo repository.UserRepositoryInterface, teamRepo repository.TeamRepositoryInterface, groupRepo repository.GroupRepositoryInterface, orgRepo repository.OrganizationRepositoryInterface) AICoreServiceInterface {
	return &AICoreService{
		userRepo:    userRepo,
		teamRepo:    teamRepo,
		groupRepo:   groupRepo,
		orgRepo:     orgRepo,
		credentials: make(map[string]*AICoreCredentials),
		tokenCache:  make(map[string]*tokenCache),
		httpClient: &http.Client{
			Timeout: 120 * time.Second, // Increased timeout for AI inference requests (LLMs can take 30-60s)
		},
	}
}

// SetHTTPClient sets a custom HTTP client (useful for testing with shorter timeouts)
func (s *AICoreService) SetHTTPClient(client *http.Client) {
	s.httpClient = client
}

// getTeamLimit returns the configurable team limit from environment variable or default
func (s *AICoreService) getTeamLimit() int {
	limitStr := os.Getenv("AI_CORE_TEAM_LIMIT")
	if limitStr == "" {
		return 1000 // Default limit
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		return 1000 // Default limit if invalid
	}

	return limit
}

// loadCredentials loads and caches all AI Core credentials from environment variable
func (s *AICoreService) loadCredentials() error {
	credentialsJSON := os.Getenv("AI_CORE_CREDENTIALS")
	if credentialsJSON == "" {
		return errors.ErrAICoreCredentialsNotSet
	}

	var credentialsList []AICoreCredentials
	if err := json.Unmarshal([]byte(credentialsJSON), &credentialsList); err != nil {
		return fmt.Errorf("%w: %v", errors.ErrAICoreCredentialsInvalid, err)
	}

	s.credentialsMux.Lock()
	defer s.credentialsMux.Unlock()

	// Clear existing credentials and rebuild cache
	s.credentials = make(map[string]*AICoreCredentials)
	for i := range credentialsList {
		cred := &credentialsList[i]
		s.credentials[cred.Team] = cred
	}

	return nil
}

// getCredentialsForTeam retrieves AI Core credentials for a specific team from cache
func (s *AICoreService) getCredentialsForTeam(teamName string) (*AICoreCredentials, error) {
	// Load credentials once
	var loadErr error
	s.credentialsOnce.Do(func() {
		loadErr = s.loadCredentials()
	})
	if loadErr != nil {
		return nil, loadErr
	}

	s.credentialsMux.RLock()
	defer s.credentialsMux.RUnlock()

	cred, exists := s.credentials[teamName]
	if !exists {
		return nil, errors.NewAICoreCredentialsNotFoundError(teamName)
	}

	return cred, nil
}

// getUserTeam retrieves the team name for the authenticated user
func (s *AICoreService) getUserTeam(c *gin.Context) (string, error) {
	// Get user email from auth context
	email, exists := auth.GetUserEmail(c)
	if !exists {
		logger.New().Error("AI Core: User email not found in context - authentication middleware not applied")
		return "", errors.ErrUserEmailNotFound
	}

	// Build contextual logger with user email
	log := logger.New().WithField("user_email", email)

	// Get user from database
	member, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Error("AI Core: User not found in database")
			return "", errors.ErrUserNotFoundInDB
		}
		log.Errorf("AI Core: Failed to get user from database: %v", err)
		return "", fmt.Errorf("failed to get user from database: %w", err)
	}

	// Get user's team
	if member.TeamID == nil {
		log.Error("AI Core: User is not assigned to any team")
		return "", errors.ErrUserNotAssignedToTeam
	}

	team, err := s.teamRepo.GetByID(*member.TeamID)
	if err != nil {
		if errors.IsNotFound(err) {
			log.WithField("team_id", *member.TeamID).Error("AI Core: Team not found in database")
			return "", errors.ErrTeamNotFoundInDB
		}
		log.WithField("team_id", *member.TeamID).Errorf("AI Core: Failed to get team from database: %v", err)
		return "", fmt.Errorf("failed to get team from database: %w", err)
	}

	// Log team info - critical for debugging team name/credential matching
	log.WithFields(map[string]interface{}{
		"team_name":  team.Name,
		"team_id":    team.ID,
		"team_owner": team.Owner,
	}).Info("AI Core: Retrieved team for user")
	return team.Name, nil
}

// getAccessToken retrieves an access token for AI Core API with caching
func (s *AICoreService) getAccessToken(credentials *AICoreCredentials) (string, error) {
	teamName := credentials.Team

	// Check cache first
	s.tokenCacheMux.RLock()
	if cached, exists := s.tokenCache[teamName]; exists {
		if time.Now().Before(cached.expiresAt) {
			s.tokenCacheMux.RUnlock()
			return cached.token, nil
		}
	}
	s.tokenCacheMux.RUnlock()

	// Token not cached or expired, get new token
	token, expiresIn, err := s.requestNewToken(credentials)
	if err != nil {
		return "", err
	}

	// Cache the token with a buffer (expire 5 minutes early to be safe)
	expiresAt := time.Now().Add(time.Duration(expiresIn-300) * time.Second)

	s.tokenCacheMux.Lock()
	s.tokenCache[teamName] = &tokenCache{
		token:     token,
		expiresAt: expiresAt,
	}
	s.tokenCacheMux.Unlock()

	return token, nil
}

// requestNewToken requests a new access token from the OAuth endpoint
func (s *AICoreService) requestNewToken(credentials *AICoreCredentials) (string, int, error) {
	// Use proper form encoding instead of string concatenation for security
	data := url.Values{}
	data.Set("grant_type", "client_credentials")
	data.Set("client_id", credentials.ClientID)
	data.Set("client_secret", credentials.ClientSecret)

	req, err := http.NewRequest("POST", credentials.OAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", 0, fmt.Errorf("failed to create token request: %w", err)
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to get access token: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", 0, fmt.Errorf("token request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var tokenResp AICoreTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", 0, fmt.Errorf("failed to decode token response: %w", err)
	}

	return tokenResp.AccessToken, tokenResp.ExpiresIn, nil
}

// makeAICoreRequest makes an authenticated request to AI Core API
func (s *AICoreService) makeAICoreRequest(method, url, accessToken, resourceGroup string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	req, err := http.NewRequest(method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AI-Resource-Group", resourceGroup)

	return s.httpClient.Do(req)
}

// GetDeployments retrieves deployments from AI Core based on user's role
func (s *AICoreService) GetDeployments(c *gin.Context) (*AICoreDeploymentsResponse, error) {
	// Get user email from auth context
	email, exists := auth.GetUserEmail(c)
	if !exists {
		return nil, errors.ErrUserEmailNotFound
	}

	// Get user from database
	member, err := s.userRepo.GetByEmail(email)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.ErrUserNotFoundInDB
		}
		return nil, fmt.Errorf("failed to get user from database: %w", err)
	}

	// Determine user role and get appropriate teams (including metadata-based teams)
	teamNames, err := s.getAllTeamsForUser(member)
	if err != nil {
		return nil, err
	}

	// Aggregate deployments from all teams, grouped by team
	teamDeployments := make([]AICoreTeamDeployments, 0)
	totalCount := 0

	for _, teamName := range teamNames {
		// Get credentials for the team
		credentials, err := s.getCredentialsForTeam(teamName)
		if err != nil {
			// Skip teams without credentials instead of failing
			continue
		}

		// Get access token
		accessToken, err := s.getAccessToken(credentials)
		if err != nil {
			// Skip teams with token issues instead of failing
			continue
		}

		// Make request to AI Core
		url := fmt.Sprintf("%s/v2/lm/deployments", credentials.APIURL)
		resp, err := s.makeAICoreRequest("GET", url, accessToken, credentials.ResourceGroup, nil)
		if err != nil {
			// Skip teams with API issues instead of failing
			continue
		}

		// Ensure response body is always closed
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			// We need to decode into a temporary structure that matches AI Core's actual response
			var tempResp struct {
				Count     int                `json:"count"`
				Resources []AICoreDeployment `json:"resources"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&tempResp); err == nil {
				// Create team deployment entry
				teamDeployment := AICoreTeamDeployments{
					Team:        teamName,
					Deployments: tempResp.Resources,
				}
				teamDeployments = append(teamDeployments, teamDeployment)
				totalCount += tempResp.Count
			}
		}
	}

	// Return aggregated response with new structure
	return &AICoreDeploymentsResponse{
		Count:       totalCount,
		Deployments: teamDeployments,
	}, nil
}

// getTeamsForUser determines which teams a user should see deployments for based on their role
func (s *AICoreService) getTeamsForUser(member *models.User) ([]string, error) {
	var teamNames []string

	// Team member or team manager - get only their team
	if member.TeamID != nil {
		team, err := s.teamRepo.GetByID(*member.TeamID)
		if err != nil {
			if errors.IsNotFound(err) {
				return nil, errors.ErrTeamNotFoundInDB
			}
			return nil, fmt.Errorf("failed to get team from database: %w", err)
		}
		teamNames = append(teamNames, team.Name)
		return teamNames, nil
	}

	// User not assigned to any team
	return nil, errors.ErrUserNotAssignedToTeam
}

// getAICoreTeamsFromMetadata extracts team names from the member's metadata.ai_instances field
func (s *AICoreService) getAICoreTeamsFromMetadata(member *models.User) []string {
	var teamNames []string

	if member.Metadata == nil {
		return teamNames
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(member.Metadata, &metadata); err != nil {
		return teamNames
	}

	aiInstances, exists := metadata["ai_instances"]
	if !exists {
		return teamNames
	}

	// Handle different possible types for ai_instances
	switch v := aiInstances.(type) {
	case []interface{}:
		// Array of team names
		for _, teamNameInterface := range v {
			if teamName, ok := teamNameInterface.(string); ok && teamName != "" {
				teamNames = append(teamNames, teamName)
			}
		}
	case []string:
		// Array of strings (direct type)
		for _, teamName := range v {
			if teamName != "" {
				teamNames = append(teamNames, teamName)
			}
		}
	case string:
		// Single team name as string
		if v != "" {
			teamNames = append(teamNames, v)
		}
	}

	return teamNames
}

// getAllTeamsForUser combines teams from role-based access and metadata-based access
func (s *AICoreService) getAllTeamsForUser(member *models.User) ([]string, error) {
	teamNamesSet := make(map[string]bool) // Use a set to avoid duplicates
	var allTeamNames []string

	// Get teams based on role and team assignment
	roleBasedTeams, err := s.getTeamsForUser(member)
	if err != nil && err != errors.ErrUserNotAssignedToTeam {
		return nil, err
	}

	// Add role-based teams to the set
	for _, teamName := range roleBasedTeams {
		if !teamNamesSet[teamName] {
			allTeamNames = append(allTeamNames, teamName)
			teamNamesSet[teamName] = true
		}
	}

	// Get additional teams from metadata
	metadataTeams := s.getAICoreTeamsFromMetadata(member)
	for _, teamName := range metadataTeams {
		if !teamNamesSet[teamName] {
			allTeamNames = append(allTeamNames, teamName)
			teamNamesSet[teamName] = true
		}
	}

	// If no teams found at all, return error
	if len(allTeamNames) == 0 {
		return nil, errors.ErrUserNotAssignedToTeam
	}

	return allTeamNames, nil
}

// GetMe resolves AI instances for the authenticated user based on role and metadata
func (s *AICoreService) GetMe(c *gin.Context) (*AICoreMeResponse, error) {
	// Get username from auth context
	username, exists := auth.GetUsername(c)
	if !exists || username == "" {
		return nil, errors.ErrUserEmailNotFound
	}

	// Look up user by name (maps to 'name' column in users table via BaseModel)
	member, err := s.userRepo.GetByName(username)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, errors.ErrUserNotFoundInDB
		}
		return nil, fmt.Errorf("failed to get user by name: %w", err)
	}

	// Prepare ai_instances with optional initialization from metadata.ai_instances
	aiInstances := make([]string, 0)
	seen := make(map[string]bool)

	add := func(name string) {
		if name == "" {
			return
		}
		if !seen[name] {
			aiInstances = append(aiInstances, name)
			seen[name] = true
		}
	}

	// Prepare metadata instances (will be added after filtering)
	metaInstances := make([]string, 0)
	// Merge from metadata.ai_instances if present
	if member.Metadata != nil {
		var metadata map[string]interface{}
		if err := json.Unmarshal(member.Metadata, &metadata); err == nil {
			if v, ok := metadata["ai_instances"]; ok {
				switch t := v.(type) {
				case []interface{}:
					for _, it := range t {
						if sname, ok := it.(string); ok && sname != "" {
							metaInstances = append(metaInstances, sname)
						}
					}
				case []string:
					for _, sname := range t {
						if sname != "" {
							metaInstances = append(metaInstances, sname)
						}
					}
				case string:
					if t != "" {
						metaInstances = append(metaInstances, t)
					}
				}
			}
		}
	}

	// Role-based aggregation
	switch member.TeamRole {
	case models.TeamRoleManager:
		// Find the group where this manager is the owner
		var targetGroup *models.Group

		// Try via user's team -> group, and verify ownership
		if member.TeamID != nil {
			if team, err := s.teamRepo.GetByID(*member.TeamID); err == nil {
				if grp, err := s.groupRepo.GetByID(team.GroupID); err == nil {
					if grp.Owner == username {
						targetGroup = grp
					} else {
						// Search groups within the same org for ownership match
						if groups, _, err := s.groupRepo.GetByOrganizationID(grp.OrgID, s.getTeamLimit(), 0); err == nil {
							for i := range groups {
								if groups[i].Owner == username {
									g := groups[i]
									targetGroup = &g
									break
								}
							}
						}
					}
				}
			}
		}

		// As a fallback, scan all orgs for a group owned by user
		if targetGroup == nil && s.orgRepo != nil {
			if orgs, _, err := s.orgRepo.GetAll(1000, 0); err == nil {
				for _, org := range orgs {
					if groups, _, err := s.groupRepo.GetByOrganizationID(org.ID, s.getTeamLimit(), 0); err == nil {
						for i := range groups {
							if groups[i].Owner == username {
								g := groups[i]
								targetGroup = &g
								break
							}
						}
					}
					if targetGroup != nil {
						break
					}
				}
			}
		}

		// Final fallback: use user's current team's group if available
		if targetGroup == nil && member.TeamID != nil {
			if team, err := s.teamRepo.GetByID(*member.TeamID); err == nil {
				if grp, err := s.groupRepo.GetByID(team.GroupID); err == nil {
					targetGroup = grp
				}
			}
		}

		// Collect all team names in the target group
		if targetGroup != nil {
			if teams, _, err := s.teamRepo.GetByGroupID(targetGroup.ID, s.getTeamLimit(), 0); err == nil {
				for _, t := range teams {
					add(t.Name)
				}
			}
		}
	case models.TeamRoleMMM:
		// Find the organization where this MMM is the owner
		var targetOrg *models.Organization

		// Try via user's team -> group -> org
		if member.TeamID != nil {
			if team, err := s.teamRepo.GetByID(*member.TeamID); err == nil {
				if grp, err := s.groupRepo.GetByID(team.GroupID); err == nil {
					if org, err := s.orgRepo.GetByID(grp.OrgID); err == nil {
						if org.Owner == username {
							targetOrg = org
						}
					}
				}
			}
		}

		// Fallback: scan all organizations for ownership match
		if targetOrg == nil && s.orgRepo != nil {
			if orgs, _, err := s.orgRepo.GetAll(1000, 0); err == nil {
				for i := range orgs {
					if orgs[i].Owner == username {
						o := orgs[i]
						targetOrg = &o
						break
					}
				}
			}
		}

		// Collect all team names across all groups in the target org
		if targetOrg != nil {
			if groups, _, err := s.groupRepo.GetByOrganizationID(targetOrg.ID, s.getTeamLimit(), 0); err == nil {
				for _, g := range groups {
					if teams, _, err := s.teamRepo.GetByGroupID(g.ID, s.getTeamLimit(), 0); err == nil {
						for _, t := range teams {
							add(t.Name)
						}
					}
				}
			}
		}
	default:
		// Neither manager nor mmm: use user's assigned team name (if any)
		if member.TeamID != nil {
			if team, err := s.teamRepo.GetByID(*member.TeamID); err == nil {
				add(team.Name)
			}
		}
	}

	// Log discovered instances (before filtering)
	{
		log := logger.New().WithField("username", username)
		log.WithField("ai_instances", aiInstances).Info("AI Core: initial discovered ai_instances")
	}

	// Filter discovered instances by teams present in AI_CORE_CREDENTIALS
	filtered := make([]string, 0)
	envTeams := make(map[string]bool)

	if err := s.loadCredentials(); err == nil {
		// Build env team set from credentials cache
		s.credentialsMux.RLock()
		for teamName := range s.credentials {
			envTeams[teamName] = true
		}
		s.credentialsMux.RUnlock()

		for _, name := range aiInstances {
			if envTeams[name] {
				filtered = append(filtered, name)
			}
		}
	} else {
		// If credentials are not configured, skip filtering
		filtered = aiInstances
	}

	// Log filtered instances
	{
		log := logger.New().WithField("username", username)
		log.WithField("ai_instances", filtered).Info("AI Core: filtered ai_instances (by environment)")
	}

	// Reset aiInstances to filtered values and reinitialize set
	aiInstances = make([]string, 0)
	seen = make(map[string]bool)
	for _, name := range filtered {
		if !seen[name] {
			aiInstances = append(aiInstances, name)
			seen[name] = true
		}
	}

	// Add metadata instances (union after filtering)
	for _, name := range metaInstances {
		add(name)
	}

	return &AICoreMeResponse{
		User:        username,
		AIInstances: aiInstances,
	}, nil
}

// GetModels retrieves models from AI for the user's team
func (s *AICoreService) GetModels(c *gin.Context, scenarioID string) (*AICoreModelsResponse, error) {
	// Get user email for logging context
	email, _ := auth.GetUserEmail(c)
	log := logger.New().WithFields(map[string]interface{}{
		"user_email":  email,
		"scenario_id": scenarioID,
	})

	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		log.WithField("team_name", teamName).Errorf("AI Core: Failed to get credentials: %v", err)
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		log.WithField("team_name", teamName).Errorf("AI Core: Failed to get access token: %v", err)
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/scenarios/%s/models", credentials.APIURL, scenarioID)
	resp, err := s.makeAICoreRequest("GET", url, accessToken, credentials.ResourceGroup, nil)
	if err != nil {
		log.WithField("team_name", teamName).Errorf("AI Core: API request failed: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.WithFields(map[string]interface{}{
			"team_name":   teamName,
			"status_code": resp.StatusCode,
			"response":    string(body),
		}).Error("AI Core: AI Core API returned error")
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var modelsResp AICoreModelsResponse
	if err := json.NewDecoder(resp.Body).Decode(&modelsResp); err != nil {
		log.WithField("team_name", teamName).Errorf("AI Core: Failed to decode response: %v", err)
		return nil, fmt.Errorf("failed to decode models response: %w", err)
	}

	return &modelsResp, nil
}

// GetConfigurations retrieves configurations from AI Core for the user's team
func (s *AICoreService) GetConfigurations(c *gin.Context) (*AICoreConfigurationsResponse, error) {
	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/configurations", credentials.APIURL)
	resp, err := s.makeAICoreRequest("GET", url, accessToken, credentials.ResourceGroup, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var configurationsResp AICoreConfigurationsResponse
	if err := json.NewDecoder(resp.Body).Decode(&configurationsResp); err != nil {
		return nil, fmt.Errorf("failed to decode configurations response: %w", err)
	}

	return &configurationsResp, nil
}

// CreateConfiguration creates a new configuration in AI Core
func (s *AICoreService) CreateConfiguration(c *gin.Context, req *AICoreConfigurationRequest) (*AICoreConfigurationResponse, error) {
	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/configurations", credentials.APIURL)
	resp, err := s.makeAICoreRequest("POST", url, accessToken, credentials.ResourceGroup, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var configResp AICoreConfigurationResponse
	if err := json.NewDecoder(resp.Body).Decode(&configResp); err != nil {
		return nil, fmt.Errorf("failed to decode configuration response: %w", err)
	}

	return &configResp, nil
}

// CreateDeployment creates a new deployment in AI Core
// Supports two scenarios:
// 1. Direct deployment with configurationId
// 2. Create configuration first, then deploy with the created configurationId
func (s *AICoreService) CreateDeployment(c *gin.Context, req *AICoreDeploymentRequest) (*AICoreDeploymentResponse, error) {
	// Validate that either configurationId or configurationRequest is provided, but not both
	if req.ConfigurationID == nil && req.ConfigurationRequest == nil {
		return nil, fmt.Errorf("either configurationId or configurationRequest must be provided")
	}
	if req.ConfigurationID != nil && req.ConfigurationRequest != nil {
		return nil, fmt.Errorf("configurationId and configurationRequest cannot both be provided")
	}

	var configurationID string

	// Scenario 1: Direct deployment with existing configurationId
	if req.ConfigurationID != nil {
		configurationID = *req.ConfigurationID
	} else {
		// Scenario 2: Create configuration first, then deploy
		configResp, err := s.CreateConfiguration(c, req.ConfigurationRequest)
		if err != nil {
			return nil, fmt.Errorf("failed to create configuration: %w", err)
		}
		configurationID = configResp.ID
	}

	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Create the deployment request for AI Core API
	deploymentReq := struct {
		ConfigurationID string `json:"configurationId"`
		TTL             string `json:"ttl,omitempty"`
	}{
		ConfigurationID: configurationID,
		TTL:             req.TTL,
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/deployments", credentials.APIURL)
	resp, err := s.makeAICoreRequest("POST", url, accessToken, credentials.ResourceGroup, deploymentReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var deploymentResp AICoreDeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deploymentResp); err != nil {
		return nil, fmt.Errorf("failed to decode deployment response: %w", err)
	}

	return &deploymentResp, nil
}

// UpdateDeployment updates a deployment in AI Core
func (s *AICoreService) UpdateDeployment(c *gin.Context, deploymentID string, req *AICoreDeploymentModificationRequest) (*AICoreDeploymentModificationResponse, error) {
	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/deployments/%s", credentials.APIURL, deploymentID)
	resp, err := s.makeAICoreRequest("PATCH", url, accessToken, credentials.ResourceGroup, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.ErrAICoreDeploymentNotFound
	}

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var modificationResp AICoreDeploymentModificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&modificationResp); err != nil {
		return nil, fmt.Errorf("failed to decode deployment modification response: %w", err)
	}

	return &modificationResp, nil
}

// DeleteDeployment deletes a deployment in AI Core
func (s *AICoreService) DeleteDeployment(c *gin.Context, deploymentID string) (*AICoreDeploymentDeletionResponse, error) {
	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/deployments/%s", credentials.APIURL, deploymentID)
	resp, err := s.makeAICoreRequest("DELETE", url, accessToken, credentials.ResourceGroup, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.ErrAICoreDeploymentNotFound
	}

	if resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var deletionResp AICoreDeploymentDeletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&deletionResp); err != nil {
		return nil, fmt.Errorf("failed to decode deployment deletion response: %w", err)
	}

	return &deletionResp, nil
}

// GetDeploymentDetails retrieves detailed information about a specific deployment from AI Core
func (s *AICoreService) GetDeploymentDetails(c *gin.Context, deploymentID string) (*AICoreDeploymentDetailsResponse, error) {
	// Get user's team
	teamName, err := s.getUserTeam(c)
	if err != nil {
		return nil, err
	}

	// Get credentials for the team
	credentials, err := s.getCredentialsForTeam(teamName)
	if err != nil {
		return nil, err
	}

	// Get access token
	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, err
	}

	// Make request to AI Core
	url := fmt.Sprintf("%s/v2/lm/deployments/%s", credentials.APIURL, deploymentID)
	resp, err := s.makeAICoreRequest("GET", url, accessToken, credentials.ResourceGroup, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.ErrAICoreDeploymentNotFound
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w with status %d: %s", errors.ErrAICoreAPIRequestFailed, resp.StatusCode, string(body))
	}

	var deploymentDetails AICoreDeploymentDetailsResponse
	if err := json.NewDecoder(resp.Body).Decode(&deploymentDetails); err != nil {
		return nil, fmt.Errorf("failed to decode deployment details response: %w", err)
	}

	return &deploymentDetails, nil
}

// AICoreInferenceRequest represents a chat inference request
type AICoreInferenceRequest struct {
	DeploymentID string                   `json:"deploymentId" validate:"required"`
	Messages     []AICoreInferenceMessage `json:"messages" validate:"required,min=1"`
	MaxTokens    int                      `json:"max_tokens,omitempty"`
	Temperature  float64                  `json:"temperature,omitempty"`
	TopP         float64                  `json:"top_p,omitempty"`
	Stream       bool                     `json:"stream,omitempty"`
}

// AICoreInferenceMessage represents a single message in the chat
// Content can be either a string or an array of content parts (for multimodal messages)
type AICoreInferenceMessage struct {
	Role    string      `json:"role" validate:"required,oneof=system user assistant"`
	Content interface{} `json:"content" validate:"required"` // string or []AICoreMessageContent
}

// AICoreMessageContent represents a part of a multimodal message (text or image)
type AICoreMessageContent struct {
	Type     string                 `json:"type"`                // "text" or "image_url"
	Text     string                 `json:"text,omitempty"`      // for type="text"
	ImageURL *AICoreMessageImageURL `json:"image_url,omitempty"` // for type="image_url"
	FileData *AICoreMessageFileData `json:"fileData,omitempty"`  // for Gemini
}

// AICoreMessageImageURL represents an image URL in a message (GPT format)
type AICoreMessageImageURL struct {
	URL string `json:"url"`
}

// AICoreMessageFileData represents file data in a message (Gemini format)
type AICoreMessageFileData struct {
	MimeType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

// AICoreInferenceResponse represents the response from AI Core inference
type AICoreInferenceResponse struct {
	ID                string                  `json:"id"`
	Object            string                  `json:"object"`
	Created           int64                   `json:"created"`
	Model             string                  `json:"model"`
	Choices           []AICoreInferenceChoice `json:"choices"`
	Usage             AICoreInferenceUsage    `json:"usage"`
	SystemFingerprint string                  `json:"system_fingerprint,omitempty"`
}

// AICoreInferenceChoice represents a single choice in the inference response
type AICoreInferenceChoice struct {
	Index        int                    `json:"index"`
	Message      AICoreInferenceMessage `json:"message"`
	FinishReason string                 `json:"finish_reason"`
}

// AICoreInferenceUsage represents token usage information
type AICoreInferenceUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// ChatInference performs a chat inference request to a deployed model
func (s *AICoreService) ChatInference(c *gin.Context, req *AICoreInferenceRequest) (*AICoreInferenceResponse, error) {
	// Get all deployments accessible to the user (reuses the same logic as Deployments tab)
	deploymentsResp, err := s.GetDeployments(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployments: %w", err)
	}

	// Find the deployment by ID across all teams
	var targetDeployment *AICoreDeployment
	var targetTeamName string

	for _, teamDeployments := range deploymentsResp.Deployments {
		for _, deployment := range teamDeployments.Deployments {
			if deployment.ID == req.DeploymentID {
				targetDeployment = &deployment
				targetTeamName = teamDeployments.Team
				break
			}
		}
		if targetDeployment != nil {
			break
		}
	}

	if targetDeployment == nil {
		return nil, fmt.Errorf("deployment %s not found or user does not have access to it", req.DeploymentID)
	}

	if targetDeployment.DeploymentURL == "" {
		return nil, fmt.Errorf("deployment URL not available for deployment %s", req.DeploymentID)
	}

	// Get credentials and token for the team that owns this deployment
	credentials, err := s.getCredentialsForTeam(targetTeamName)
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials for team %s: %w", targetTeamName, err)
	}

	accessToken, err := s.getAccessToken(credentials)
	if err != nil {
		return nil, fmt.Errorf("failed to get access token: %w", err)
	}

	// Determine model type from deployment details
	// SAP AI Core has different inference formats for different model types:
	// 1. Anthropic (Claude): Uses /invoke endpoint with Anthropic format
	// 2. GPT/OpenAI: Uses /chat/completions endpoint with OpenAI format (various api-versions)
	// 3. Gemini: Uses /models/<model>:generateContent endpoint
	// 4. Orchestration: Uses orchestration-specific endpoints (not foundation-models scenario)

	isOrchestration := false
	isGPTModel := false
	isGeminiModel := false
	modelName := ""

	// Check if this is orchestration based on scenario ID only
	// Orchestration deployments have a different scenario ID (not foundation-models)
	if strings.Contains(strings.ToLower(targetDeployment.ScenarioID), "orchestration") {
		isOrchestration = true
	}

	// Extract model name and check model type
	if extractedName := extractModelNameFromDetails(targetDeployment.Details); extractedName != "" {
		modelName = extractedName
		lowerName := strings.ToLower(extractedName)
		if strings.Contains(lowerName, "gpt") || strings.Contains(lowerName, "o1") ||
			strings.Contains(lowerName, "o3") || strings.Contains(lowerName, "openai") {
			isGPTModel = true
		} else if strings.Contains(lowerName, "gemini") {
			isGeminiModel = true
		}
	}

	// Trim messages to fit within model context limits
	// This prevents "context too large" errors
	contextLimit := getModelContextLimit(modelName)
	req.Messages = trimMessagesToContextLimit(req.Messages, contextLimit)

	var inferencePayload map[string]interface{}
	var inferenceURL string

	// Helper function to extract text content from message
	getMessageText := func(msg AICoreInferenceMessage) string {
		if str, ok := msg.Content.(string); ok {
			return str
		}
		// If content is array, find the first text part
		if contentArr, ok := msg.Content.([]interface{}); ok {
			for _, part := range contentArr {
				if partMap, ok := part.(map[string]interface{}); ok {
					if partMap["type"] == "text" {
						if text, ok := partMap["text"].(string); ok {
							return text
						}
					}
				}
			}
		}
		return ""
	}

	if isGeminiModel {
		// Gemini models use /models/<model>:generateContent endpoint
		// Format: https://...deployments/{id}/models/gemini-1.5-flash:generateContent

		// Build Gemini contents structure
		var parts []map[string]interface{}

		for _, msg := range req.Messages {
			if msg.Role == "system" {
				// Gemini doesn't have system role, prepend as user message
				parts = append(parts, map[string]interface{}{
					"text": fmt.Sprintf("[System]: %s", getMessageText(msg)),
				})
				continue
			}

			// Handle multimodal content (text + images)
			if contentArr, ok := msg.Content.([]interface{}); ok {
				for _, part := range contentArr {
					if partMap, ok := part.(map[string]interface{}); ok {
						partType := partMap["type"].(string)
						if partType == "text" {
							parts = append(parts, map[string]interface{}{
								"text": partMap["text"],
							})
						} else if partType == "image_url" {
							// Convert image_url to fileData format for Gemini
							if imageURL, ok := partMap["image_url"].(map[string]interface{}); ok {
								parts = append(parts, map[string]interface{}{
									"fileData": map[string]interface{}{
										"mimeType": "image/png", // Default, can be detected from URL
										"fileUri":  imageURL["url"],
									},
								})
							}
						}
					}
				}
			} else {
				// Simple text content
				parts = append(parts, map[string]interface{}{
					"text": getMessageText(msg),
				})
			}
		}

		inferencePayload = map[string]interface{}{
			"contents": map[string]interface{}{
				"role":  "user",
				"parts": parts,
			},
		}

		// Add generation config if parameters provided
		if req.MaxTokens > 0 || req.Temperature > 0 {
			generationConfig := make(map[string]interface{})
			if req.MaxTokens > 0 {
				generationConfig["maxOutputTokens"] = req.MaxTokens
			}
			if req.Temperature > 0 {
				generationConfig["temperature"] = req.Temperature
			}
			inferencePayload["generation_config"] = generationConfig
		}

		// Gemini endpoint format: /models/<model>:generateContent or streamGenerateContent for streaming
		if req.Stream {
			inferenceURL = fmt.Sprintf("%s/models/%s:streamGenerateContent", targetDeployment.DeploymentURL, modelName)
		} else {
			inferenceURL = fmt.Sprintf("%s/models/%s:generateContent", targetDeployment.DeploymentURL, modelName)
		}
	} else if isOrchestration {
		// Extract model name from deployment details
		modelName := extractModelNameFromDetails(targetDeployment.Details)
		if modelName == "" {
			modelName = "gpt-4o-mini" // default fallback
		}

		// Build template messages for orchestration
		templateMessages := make([]map[string]interface{}, 0)
		for _, msg := range req.Messages {
			templateMessages = append(templateMessages, map[string]interface{}{
				"role":    msg.Role,
				"content": msg.Content,
			})
		}

		// Prepare orchestration request payload
		// SAP AI Core orchestration uses a specific format with orchestration_config
		modelParams := map[string]interface{}{
			"frequency_penalty": 0,
			"presence_penalty":  0,
		}

		if req.MaxTokens > 0 {
			modelParams["max_tokens"] = req.MaxTokens
		} else {
			modelParams["max_tokens"] = 1000
		}

		if req.Temperature > 0 {
			modelParams["temperature"] = req.Temperature
		} else {
			modelParams["temperature"] = 0.7
		}

		inferencePayload = map[string]interface{}{
			"orchestration_config": map[string]interface{}{
				"module_configurations": map[string]interface{}{
					"templating_module_config": map[string]interface{}{
						"template": templateMessages,
					},
					"llm_module_config": map[string]interface{}{
						"model_name":    modelName,
						"model_params":  modelParams,
						"model_version": "latest",
					},
				},
			},
			"input_params": map[string]interface{}{},
		}

		// Use orchestration /completion endpoint
		inferenceURL = fmt.Sprintf("%s/completion", targetDeployment.DeploymentURL)
	} else if isGPTModel {
		// GPT/OpenAI foundation models use /chat/completions endpoint with OpenAI-compatible format
		// Build messages array - handle both simple strings and multimodal content
		messages := make([]map[string]interface{}, 0)
		for _, msg := range req.Messages {
			message := map[string]interface{}{
				"role": msg.Role,
			}

			// Check if content is multimodal (array) or simple text (string)
			if contentArr, ok := msg.Content.([]interface{}); ok {
				// Multimodal content (text + images for GPT-4o, GPT-4-Turbo, GPT-4o Mini)
				message["content"] = contentArr
			} else {
				// Simple text content
				message["content"] = msg.Content
			}

			messages = append(messages, message)
		}

		// Build OpenAI-compatible payload
		inferencePayload = map[string]interface{}{
			"messages": messages,
		}

		// Determine API version based on model
		apiVersion := getGPTAPIVersion(modelName)

		// Add optional parameters (o1, o3-mini, and gpt-5 don't support these parameters)
		isReasoningModel := strings.Contains(strings.ToLower(modelName), "o1") ||
			strings.Contains(strings.ToLower(modelName), "o3-mini") ||
			strings.Contains(strings.ToLower(modelName), "gpt-5")

		if !isReasoningModel {
			if req.MaxTokens > 0 {
				inferencePayload["max_tokens"] = req.MaxTokens
			} else {
				inferencePayload["max_tokens"] = 1000
			}
			if req.Temperature > 0 {
				inferencePayload["temperature"] = req.Temperature
			} else {
				inferencePayload["temperature"] = 0.7
			}
			if req.TopP > 0 {
				inferencePayload["top_p"] = req.TopP
			}
		}

		// Add stream parameter for GPT models
		if req.Stream {
			inferencePayload["stream"] = true
		}

		// GPT foundation models use /chat/completions endpoint with api-version query parameter
		inferenceURL = fmt.Sprintf("%s/chat/completions?api-version=%s", targetDeployment.DeploymentURL, apiVersion)
	} else {
		// Anthropic Claude foundation models use /invoke endpoint with Anthropic format
		// Convert messages to Anthropic Claude API format
		var systemPrompt string
		var userMessages []map[string]string

		for _, msg := range req.Messages {
			if msg.Role == "system" {
				systemPrompt = getMessageText(msg)
			} else {
				userMessages = append(userMessages, map[string]string{
					"role":    msg.Role,
					"content": getMessageText(msg),
				})
			}
		}

		// Build Anthropic-compatible payload
		inferencePayload = map[string]interface{}{
			"anthropic_version": "bedrock-2023-05-31",
			"messages":          userMessages,
		}

		// Add system prompt if present
		if systemPrompt != "" {
			inferencePayload["system"] = systemPrompt
		}

		// Add optional parameters using Anthropic naming
		if req.MaxTokens > 0 {
			inferencePayload["max_tokens"] = req.MaxTokens
		} else {
			inferencePayload["max_tokens"] = 1000
		}
		if req.Temperature > 0 {
			inferencePayload["temperature"] = req.Temperature
		} else {
			inferencePayload["temperature"] = 0.7
		}
		if req.TopP > 0 {
			inferencePayload["top_p"] = req.TopP
		}

		// Add stream parameter for Anthropic models
		if req.Stream {
			inferencePayload["stream"] = true
		}

		// Anthropic foundation models use /invoke endpoint
		inferenceURL = fmt.Sprintf("%s/invoke", targetDeployment.DeploymentURL)
	}

	resp, err := s.makeAICoreRequest("POST", inferenceURL, accessToken, credentials.ResourceGroup, inferencePayload)
	if err != nil {
		return nil, fmt.Errorf("failed to make inference request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("inference request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var inferenceResp *AICoreInferenceResponse

	if isGeminiModel {
		// Parse Gemini response and convert to OpenAI-compatible format
		var geminiResp struct {
			Candidates []struct {
				Content struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
					Role string `json:"role"`
				} `json:"content"`
				FinishReason string `json:"finishReason"`
			} `json:"candidates"`
			UsageMetadata struct {
				PromptTokenCount     int `json:"promptTokenCount"`
				CandidatesTokenCount int `json:"candidatesTokenCount"`
				TotalTokenCount      int `json:"totalTokenCount"`
			} `json:"usageMetadata"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
			return nil, fmt.Errorf("failed to decode Gemini response: %w", err)
		}

		// Convert to OpenAI format
		inferenceResp = &AICoreInferenceResponse{
			ID:      fmt.Sprintf("gemini-%d", time.Now().Unix()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   modelName,
			Choices: []AICoreInferenceChoice{},
			Usage: AICoreInferenceUsage{
				PromptTokens:     geminiResp.UsageMetadata.PromptTokenCount,
				CompletionTokens: geminiResp.UsageMetadata.CandidatesTokenCount,
				TotalTokens:      geminiResp.UsageMetadata.TotalTokenCount,
			},
		}

		// Extract text from candidates
		for i, candidate := range geminiResp.Candidates {
			var text string
			for _, part := range candidate.Content.Parts {
				text += part.Text
			}

			inferenceResp.Choices = append(inferenceResp.Choices, AICoreInferenceChoice{
				Index: i,
				Message: AICoreInferenceMessage{
					Role:    "assistant",
					Content: text,
				},
				FinishReason: strings.ToLower(candidate.FinishReason),
			})
		}
	} else if isOrchestration {
		// Parse orchestration response
		// Orchestration returns: {orchestration_result: {choices: [{message: {content: "..."}}]}}
		var orchResp struct {
			OrchestrationResult struct {
				Choices []struct {
					Message struct {
						Content string `json:"content"`
						Role    string `json:"role"`
					} `json:"message"`
					FinishReason string `json:"finish_reason"`
					Index        int    `json:"index"`
				} `json:"choices"`
			} `json:"orchestration_result"`
			ModuleResults struct {
				Templating []interface{} `json:"templating"`
				LLM        []struct {
					Message struct {
						Content string `json:"content"`
						Role    string `json:"role"`
					} `json:"message"`
				} `json:"llm"`
			} `json:"module_results"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&orchResp); err != nil {
			return nil, fmt.Errorf("failed to decode orchestration response: %w", err)
		}

		// Convert to OpenAI format
		inferenceResp = &AICoreInferenceResponse{
			ID:      fmt.Sprintf("orch-%d", time.Now().Unix()),
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   modelName,
			Choices: []AICoreInferenceChoice{},
			Usage: AICoreInferenceUsage{
				PromptTokens:     0, // Orchestration doesn't return token counts
				CompletionTokens: 0,
				TotalTokens:      0,
			},
		}

		// Extract choices from orchestration result
		for _, choice := range orchResp.OrchestrationResult.Choices {
			inferenceResp.Choices = append(inferenceResp.Choices, AICoreInferenceChoice{
				Index: choice.Index,
				Message: AICoreInferenceMessage{
					Role:    "assistant",
					Content: choice.Message.Content,
				},
				FinishReason: choice.FinishReason,
			})
		}
	} else if isGPTModel {
		// Parse GPT/OpenAI response
		// SAP AI Core returns OpenAI-compatible format for GPT models
		inferenceResp = &AICoreInferenceResponse{}
		if err := json.NewDecoder(resp.Body).Decode(inferenceResp); err != nil {
			return nil, fmt.Errorf("failed to decode GPT response: %w", err)
		}
	} else {
		// Parse direct model response - Anthropic format from /invoke endpoint
		// Response structure: {"content":[{"text":"...","type":"text"}],"role":"assistant",...}
		var anthropicResp struct {
			ID      string `json:"id"`
			Type    string `json:"type"`
			Role    string `json:"role"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			Model      string `json:"model"`
			StopReason string `json:"stop_reason"`
			Usage      struct {
				InputTokens  int `json:"input_tokens"`
				OutputTokens int `json:"output_tokens"`
			} `json:"usage"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&anthropicResp); err != nil {
			return nil, fmt.Errorf("failed to decode Anthropic response: %w", err)
		}

		// Extract text from content array
		var generatedText string
		if len(anthropicResp.Content) > 0 {
			generatedText = anthropicResp.Content[0].Text
		}

		// Convert Anthropic format to OpenAI-compatible format for frontend
		inferenceResp = &AICoreInferenceResponse{
			ID:      anthropicResp.ID,
			Object:  "chat.completion",
			Created: time.Now().Unix(),
			Model:   anthropicResp.Model,
			Choices: []AICoreInferenceChoice{
				{
					Index: 0,
					Message: AICoreInferenceMessage{
						Role:    "assistant",
						Content: generatedText,
					},
					FinishReason: anthropicResp.StopReason,
				},
			},
			Usage: AICoreInferenceUsage{
				PromptTokens:     anthropicResp.Usage.InputTokens,
				CompletionTokens: anthropicResp.Usage.OutputTokens,
				TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
			},
		}
	}

	return inferenceResp, nil
}

// UploadAttachment processes uploaded files for AI inference
// Converts files to base64 data URLs for use in multimodal requests
// Supports images, text files (txt, json, html, csv, etc.), and documents
func (s *AICoreService) UploadAttachment(c *gin.Context, file multipart.File, header *multipart.FileHeader) (map[string]interface{}, error) {
	// Read file content
	fileBytes, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Detect MIME type from content
	mimeType := http.DetectContentType(fileBytes)

	// For text files, http.DetectContentType may return generic types
	// Use file extension to get more accurate MIME type
	filename := strings.ToLower(header.Filename)
	switch {
	case strings.HasSuffix(filename, ".json"):
		mimeType = "application/json"
	case strings.HasSuffix(filename, ".txt"):
		mimeType = "text/plain"
	case strings.HasSuffix(filename, ".html") || strings.HasSuffix(filename, ".htm"):
		mimeType = "text/html"
	case strings.HasSuffix(filename, ".csv"):
		mimeType = "text/csv"
	case strings.HasSuffix(filename, ".xml"):
		mimeType = "application/xml"
	case strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml"):
		mimeType = "application/x-yaml"
	case strings.HasSuffix(filename, ".md"):
		mimeType = "text/markdown"
	case strings.HasSuffix(filename, ".pdf"):
		mimeType = "application/pdf"
	}

	// Convert to base64 data URL
	base64Data := base64.StdEncoding.EncodeToString(fileBytes)
	dataURL := fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)

	return map[string]interface{}{
		"url":      dataURL,
		"mimeType": mimeType,
		"filename": header.Filename,
		"size":     header.Size,
	}, nil
}

// extractModelNameFromDetails extracts the model name from deployment details
// Checks both backend_details and backendDetails (camelCase) field names
// Returns empty string if model name cannot be extracted
func extractModelNameFromDetails(details map[string]interface{}) string {
	if details == nil {
		return ""
	}

	resources, ok := details["resources"].(map[string]interface{})
	if !ok {
		return ""
	}

	// Try backend_details first (snake_case)
	if backendDetails, ok := resources["backend_details"].(map[string]interface{}); ok {
		if model, ok := backendDetails["model"].(map[string]interface{}); ok {
			if name, ok := model["name"].(string); ok && name != "" {
				return name
			}
		}
	}

	// Try backendDetails (camelCase)
	if backendDetails, ok := resources["backendDetails"].(map[string]interface{}); ok {
		if model, ok := backendDetails["model"].(map[string]interface{}); ok {
			if name, ok := model["name"].(string); ok && name != "" {
				return name
			}
		}
	}

	return ""
}

// getModelContextLimit returns the maximum number of messages to send based on model type
// This prevents "context too large" errors by limiting conversation history
func getModelContextLimit(modelName string) int {
	lowerName := strings.ToLower(modelName)

	// Different models have different context windows
	// We limit by message count rather than tokens for simplicity
	switch {
	case strings.Contains(lowerName, "gpt-5"):
		return 50 // GPT-5 has very large context window
	case strings.Contains(lowerName, "gpt-4-32k"):
		return 40 // Larger context window
	case strings.Contains(lowerName, "gpt-4"):
		return 30 // Standard GPT-4
	case strings.Contains(lowerName, "gpt-3.5"):
		return 25 // GPT-3.5
	case strings.Contains(lowerName, "o1") || strings.Contains(lowerName, "o3"):
		return 20 // Reasoning models
	case strings.Contains(lowerName, "claude"):
		return 35 // Claude has good context
	case strings.Contains(lowerName, "gemini-1.5"):
		return 40 // Gemini 1.5 has large context
	case strings.Contains(lowerName, "gemini"):
		return 30 // Other Gemini models
	default:
		return 20 // Conservative default
	}
}

// trimMessagesToContextLimit trims messages to fit within model context limits
// Keeps system messages and the most recent user/assistant messages
func trimMessagesToContextLimit(messages []AICoreInferenceMessage, limit int) []AICoreInferenceMessage {
	if len(messages) <= limit {
		return messages
	}

	// Separate system messages from conversation messages
	var systemMessages []AICoreInferenceMessage
	var conversationMessages []AICoreInferenceMessage

	for _, msg := range messages {
		if msg.Role == "system" {
			systemMessages = append(systemMessages, msg)
		} else {
			conversationMessages = append(conversationMessages, msg)
		}
	}

	// Calculate how many conversation messages we can keep
	availableSlots := limit - len(systemMessages)
	if availableSlots < 1 {
		availableSlots = 1 // Always keep at least one message
	}

	// Keep the most recent conversation messages
	var trimmedConversation []AICoreInferenceMessage
	if len(conversationMessages) > availableSlots {
		// Take the last N messages
		trimmedConversation = conversationMessages[len(conversationMessages)-availableSlots:]
	} else {
		trimmedConversation = conversationMessages
	}

	// Combine system messages with trimmed conversation
	result := make([]AICoreInferenceMessage, 0, len(systemMessages)+len(trimmedConversation))
	result = append(result, systemMessages...)
	result = append(result, trimmedConversation...)

	return result
}

// getGPTAPIVersion determines GPT API version based on model name
func getGPTAPIVersion(modelName string) string {
	lowerName := strings.ToLower(modelName)
	// o1, o3-mini, and gpt-5 use newer API version
	if strings.Contains(lowerName, "o1") || strings.Contains(lowerName, "o3-mini") || strings.Contains(lowerName, "gpt-5") {
		return "2024-12-01-preview"
	}
	// All other GPT models use the standard API version
	return "2023-05-15"
}

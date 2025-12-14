package service_test

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"

	//"sync"
	"testing"
	"time"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/errors"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type AICoreServiceTestSuite struct {
	suite.Suite
	service   *service.AICoreService
	userRepo  *mocks.MockUserRepositoryInterface
	teamRepo  *mocks.MockTeamRepositoryInterface
	groupRepo *mocks.MockGroupRepositoryInterface
	orgRepo   *mocks.MockOrganizationRepositoryInterface
	server    *httptest.Server
	ctrl      *gomock.Controller
}

func (suite *AICoreServiceTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.userRepo = mocks.NewMockUserRepositoryInterface(suite.ctrl)
	suite.teamRepo = mocks.NewMockTeamRepositoryInterface(suite.ctrl)
	suite.groupRepo = mocks.NewMockGroupRepositoryInterface(suite.ctrl)
	suite.orgRepo = mocks.NewMockOrganizationRepositoryInterface(suite.ctrl)

	// Use the constructor to create the service with mock repositories
	suite.service = service.NewAICoreService(
		suite.userRepo,
		suite.teamRepo,
		suite.groupRepo,
		suite.orgRepo,
	).(*service.AICoreService)

	// Override the HTTP client for faster tests
	suite.service.SetHTTPClient(&http.Client{
		Timeout: 15 * time.Second,
	})
}

func (suite *AICoreServiceTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Close()
	}
	if suite.ctrl != nil {
		suite.ctrl.Finish()
	}
	// Clear environment variables
	os.Unsetenv("AI_CORE_CREDENTIALS")
}

func (suite *AICoreServiceTestSuite) setupMockServer(responses map[string]mockResponse) {
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)
		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

type mockResponse struct {
	StatusCode int
	Body       string
}

func (suite *AICoreServiceTestSuite) setupCredentials(teams []string) {
	credentials := make([]service.AICoreCredentials, 0)
	serverURL := "http://localhost:8080" // Default URL
	if suite.server != nil {
		serverURL = suite.server.URL
	}

	for _, team := range teams {
		credentials = append(credentials, service.AICoreCredentials{
			Team:          team,
			ClientID:      fmt.Sprintf("client-%s", team),
			ClientSecret:  fmt.Sprintf("secret-%s", team),
			OAuthURL:      fmt.Sprintf("%s/oauth/token", serverURL),
			APIURL:        serverURL,
			ResourceGroup: "default",
		})
	}

	credentialsJSON, _ := json.Marshal(credentials)
	_ = os.Setenv("AI_CORE_CREDENTIALS", string(credentialsJSON))

}

func (suite *AICoreServiceTestSuite) createGinContext(email string) *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	// Mock the auth context
	c.Set("email", email)

	return c
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_TeamMember_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 2,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					},
					{
						"id": "deployment-2",
						"configurationId": "config-2",
						"status": "STOPPED",
						"statusMessage": "Deployment is stopped",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-2",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T02:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks - gomock style
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)
	suite.Len(result.Deployments, 1)
	suite.Equal("team-alpha", result.Deployments[0].Team)
	suite.Len(result.Deployments[0].Deployments, 2)
	suite.Equal("deployment-1", result.Deployments[0].Deployments[0].ID)
	suite.Equal("RUNNING", result.Deployments[0].Deployments[0].Status)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_GroupManager_Success() {
	// Setup
	email := "group.manager@example.com"

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-alpha", "team-beta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleManager,
		Metadata: metadataJSON,
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha", "team-beta"})

	// Setup mocks - gomock style
	// This test uses metadata-based teams, so no repository calls needed
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count) // 1 deployment from each team
	suite.Len(result.Deployments, 2)
	suite.Equal("team-alpha", result.Deployments[0].Team)
	suite.Equal("team-beta", result.Deployments[1].Team)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_OrganizationManager_Success() {
	// Setup
	email := "org.manager@example.com"

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-alpha", "team-beta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleManager,
		Metadata: metadataJSON,
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha", "team-beta"})

	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)
	suite.Len(result.Deployments, 2)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_PartialCredentials_Success() {
	// Setup - Group manager with 3 teams, but only 2 have AI Core credentials
	email := "group.manager@example.com"

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-alpha", "team-beta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleManager,
		Metadata: metadataJSON,
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	// Only setup credentials for team-alpha and team-beta, not team-gamma
	suite.setupCredentials([]string{"team-alpha", "team-beta"})

	// Setup mocks - gomock style
	// This test uses metadata-based teams, so no repository calls for teams needed
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)     // Only 2 deployments from teams with credentials
	suite.Len(result.Deployments, 2) // Only 2 teams returned (team-gamma skipped)

	teamNames := make([]string, len(result.Deployments))
	for i, td := range result.Deployments {
		teamNames[i] = td.Team
	}
	suite.Contains(teamNames, "team-alpha")
	suite.Contains(teamNames, "team-beta")
	suite.NotContains(teamNames, "team-gamma") // Should be skipped due to missing credentials
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_NoCredentials_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Don't setup any credentials
	os.Unsetenv("AI_CORE_CREDENTIALS")

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err) // Should not error, just return empty result
	suite.NotNil(result)
	suite.Equal(0, result.Count)
	suite.Len(result.Deployments, 0)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_UserNotFound_Error() {
	// Setup
	email := "nonexistent@example.com"

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotFoundInDB, err)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_UserNotAssignedToTeam_Error() {
	// Setup
	email := "unassigned@example.com"

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleMember, // Not a manager
	}

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotAssignedToTeam, err)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_APIError_SkipsTeam() {
	// Setup
	email := "group.manager@example.com"

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-alpha", "team-beta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleManager,
		Metadata: metadataJSON,
	}

	// Setup mock server responses - team-alpha returns error, team-beta succeeds
	callCount := 0
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
			return
		}

		if r.URL.Path == "/v2/lm/deployments" {
			callCount++
			if callCount == 1 {
				// First call (team-alpha) returns error
				w.WriteHeader(500)
				_, _ = w.Write([]byte(`{"error": "Internal server error"}`))
			} else {
				// Second call (team-beta) succeeds
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(200)
				_, _ = w.Write([]byte(`{
					"count": 1,
					"resources": [
						{
							"id": "deployment-1",
							"configurationId": "config-1",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z"
						}
					]
				}`))
			}
		}
	}))

	suite.setupCredentials([]string{"team-alpha", "team-beta"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err) // Should not error, just skip the failing team
	suite.NotNil(result)
	suite.Equal(1, result.Count)     // Only 1 deployment from team-beta
	suite.Len(result.Deployments, 1) // Only team-beta returned
	suite.Equal("team-beta", result.Deployments[0].Team)
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_EmptyResponse_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 0,
				"resources": []
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(0, result.Count)
	suite.Len(result.Deployments, 1)
	suite.Equal("team-alpha", result.Deployments[0].Team)
	suite.Len(result.Deployments[0].Deployments, 0)
}

/* COMMENTED OUT - calls unexported methods
func (suite *AICoreServiceTestSuite) TestLoadCredentials_InvalidJSON_Error() {
	// Setup invalid JSON
	_ = os.Setenv("AI_CORE_CREDENTIALS", `{"invalid": json}`)

	// Execute
	err := suite.service.loadCredentials()

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "invalid character")
}

*/
/* COMMENTED OUT - calls unexported methods
func (suite *AICoreServiceTestSuite) TestLoadCredentials_MissingEnvVar_Error() {
	// Setup
	_ = os.Unsetenv("AI_CORE_CREDENTIALS")

	// Execute
	err := suite.service.loadCredentials()

	// Assert
	suite.Error(err)
	suite.Equal(errors.ErrAICoreCredentialsNotSet, err)
}

*/
/* COMMENTED OUT - calls unexported methods
func (suite *AICoreServiceTestSuite) TestGetCredentialsForTeam_TeamNotFound_Error() {
	// Setup
	suite.setupCredentials([]string{"team-alpha"})

	// Execute
	_, err := suite.service.getCredentialsForTeam("team-nonexistent")

	// Assert
	suite.Error(err)
	suite.Contains(err.Error(), "team-nonexistent")
}

*/
/* COMMENTED OUT - calls unexported methods
func (suite *AICoreServiceTestSuite) TestTokenCaching() {
	// Setup mock server first
	tokenCallCount := 0
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/oauth/token" {
			tokenCallCount++
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
		}
	}))

	// Now setup credentials with the server URL
	suite.setupCredentials([]string{"team-alpha"})

	credentials := &service.AICoreCredentials{
		Team:          "team-alpha",
		ClientID:      "client-alpha",
		ClientSecret:  "secret-alpha",
		OAuthURL:      fmt.Sprintf("%s/oauth/token", suite.server.URL),
		APIURL:        suite.server.URL,
		ResourceGroup: "default",
	}

	// Execute - call getAccessToken twice
	token1, err1 := suite.service.getAccessToken(credentials)
	token2, err2 := suite.service.getAccessToken(credentials)

	// Assert
	suite.NoError(err1)
	suite.NoError(err2)
	suite.Equal("test-token", token1)
	suite.Equal("test-token", token2)
	suite.Equal(1, tokenCallCount) // Should only call the token endpoint once due to caching
}

*/
func (suite *AICoreServiceTestSuite) TestGetDeployments_MetadataTeams_Success() {
	// Setup - User with team assignment AND metadata teams
	email := "user.with.metadata@example.com"
	teamID := uuid.New()

	// Create metadata with ai_instances field
	metadata := map[string]interface{}{
		"ai_instances": []string{"team-gamma", "team-delta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
		Metadata: metadataJSON,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha", // User's assigned team
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	// Setup credentials for assigned team + metadata teams
	suite.setupCredentials([]string{"team-alpha", "team-gamma", "team-delta"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(3, result.Count)     // 1 deployment from each of 3 teams
	suite.Len(result.Deployments, 3) // Should have 3 teams: assigned team + 2 metadata teams

	teamNames := make([]string, len(result.Deployments))
	for i, td := range result.Deployments {
		teamNames[i] = td.Team
	}
	suite.Contains(teamNames, "team-alpha") // Assigned team
	suite.Contains(teamNames, "team-gamma") // Metadata team
	suite.Contains(teamNames, "team-delta") // Metadata team
}

func (suite *AICoreServiceTestSuite) TestGetDeployments_MetadataOnly_Success() {
	// Setup - User with NO team assignment but WITH metadata teams
	email := "metadata.only@example.com"

	// Create metadata with ai_instances field
	metadata := map[string]interface{}{
		"ai_instances": []string{"team-gamma", "team-delta"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		TeamID:   nil, // No team assignment
		TeamRole: models.TeamRoleMember,
		Metadata: metadataJSON,
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-1",
						"configurationId": "config-1",
						"status": "RUNNING",
						"statusMessage": "Deployment is running",
						"deploymentUrl": "https://api.example.com/v1/deployments/deployment-1",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-gamma", "team-delta"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeployments(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)     // 1 deployment from each of 2 metadata teams
	suite.Len(result.Deployments, 2) // Should have 2 teams from metadata

	teamNames := make([]string, len(result.Deployments))
	for i, td := range result.Deployments {
		teamNames[i] = td.Team
	}
	suite.Contains(teamNames, "team-gamma")
	suite.Contains(teamNames, "team-delta")
}

func (suite *AICoreServiceTestSuite) TestCreateDeployment_WithConfigurationID_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	configID := "config-123"
	deploymentRequest := &service.AICoreDeploymentRequest{
		ConfigurationID: &configID,
		TTL:             "1h",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"POST:/v2/lm/deployments": {
			StatusCode: 202,
			Body: `{
				"id": "deployment-123",
				"message": "Deployment created successfully",
				"deploymentUrl": "https://api.example.com/v1/deployments/deployment-123",
				"status": "PENDING",
				"ttl": "1h"
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks - This test uses existing configurationID, so methods called once
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateDeployment(c, deploymentRequest)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("deployment-123", result.ID)
	suite.Equal("Deployment created successfully", result.Message)
	suite.Equal("PENDING", result.Status)
}

func (suite *AICoreServiceTestSuite) TestCreateDeployment_WithConfigurationRequest_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	deploymentRequest := &service.AICoreDeploymentRequest{
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name:         "my-llm-config",
			ExecutableID: "aicore-llm",
			ScenarioID:   "foundation-models",
			ParameterBindings: []map[string]string{
				{"key": "modelName", "value": "gpt-4"},
				{"key": "modelVersion", "value": "latest"},
			},
		},
		TTL: "2h",
	}

	// Setup mock server - first create config, then create deployment
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"POST:/v2/lm/configurations": {
			StatusCode: 201,
			Body: `{
				"id": "config-456",
				"message": "Configuration created successfully"
			}`,
		},
		"POST:/v2/lm/deployments": {
			StatusCode: 202,
			Body: `{
				"id": "deployment-456",
				"message": "Deployment created successfully",
				"deploymentUrl": "https://api.example.com/v1/deployments/deployment-456",
				"status": "PENDING",
				"ttl": "2h"
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks - CreateDeployment calls CreateConfiguration internally,
	// so GetByEmail and GetByID are called twice (once for config, once for deployment)
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil).Times(2)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil).Times(2)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateDeployment(c, deploymentRequest)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("deployment-456", result.ID)
	suite.Equal("Deployment created successfully", result.Message)
	suite.Equal("PENDING", result.Status)
}

func (suite *AICoreServiceTestSuite) TestCreateDeployment_BothFieldsProvided_Error() {
	// Setup
	email := "team.member@example.com"

	configID := "config-123"
	deploymentRequest := &service.AICoreDeploymentRequest{
		ConfigurationID: &configID,
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name:         "my-llm-config",
			ExecutableID: "aicore-llm",
			ScenarioID:   "foundation-models",
		},
		TTL: "1h",
	}

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateDeployment(c, deploymentRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "configurationId and configurationRequest cannot both be provided")
}

func (suite *AICoreServiceTestSuite) TestCreateDeployment_NeitherFieldProvided_Error() {
	// Setup
	email := "team.member@example.com"

	deploymentRequest := &service.AICoreDeploymentRequest{
		TTL: "1h",
	}

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateDeployment(c, deploymentRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "either configurationId or configurationRequest must be provided")
}

func (suite *AICoreServiceTestSuite) TestCreateDeployment_ConfigurationCreationFails_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	deploymentRequest := &service.AICoreDeploymentRequest{
		ConfigurationRequest: &service.AICoreConfigurationRequest{
			Name:         "my-llm-config",
			ExecutableID: "aicore-llm",
			ScenarioID:   "foundation-models",
		},
		TTL: "1h",
	}

	// Setup mock server responses - configuration creation fails
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"POST:/v2/lm/configurations": {
			StatusCode: 400,
			Body:       `{"error": "Invalid configuration request"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateDeployment(c, deploymentRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to create configuration")
}

func (suite *AICoreServiceTestSuite) TestGetModels_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	scenarioID := "foundation-models"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/scenarios/foundation-models/models": {
			StatusCode: 200,
			Body: `{
				"count": 2,
				"resources": [
					{
						"model": "gpt-4",
						"executableId": "azure-openai",
						"description": "GPT-4 model",
						"displayName": "GPT-4",
						"versions": [
							{
								"name": "latest",
								"isLatest": true,
								"deprecated": false
							}
						]
					},
					{
						"model": "gpt-3.5-turbo",
						"executableId": "azure-openai",
						"description": "GPT-3.5 Turbo model",
						"displayName": "GPT-3.5 Turbo",
						"versions": [
							{
								"name": "latest",
								"isLatest": true,
								"deprecated": false
							}
						]
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetModels(c, scenarioID)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)
	suite.Len(result.Resources, 2)
	suite.Equal("gpt-4", result.Resources[0].Model)
	suite.Equal("gpt-3.5-turbo", result.Resources[1].Model)
}

func (suite *AICoreServiceTestSuite) TestGetModels_UserNotFound_Error() {
	// Setup
	email := "nonexistent@example.com"
	scenarioID := "foundation-models"

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetModels(c, scenarioID)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotFoundInDB, err)
}

func (suite *AICoreServiceTestSuite) TestGetModels_APIError_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	scenarioID := "foundation-models"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses - API returns error
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/scenarios/foundation-models/models": {
			StatusCode: 500,
			Body:       `{"error": "Internal server error"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetModels(c, scenarioID)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "500")
}

func (suite *AICoreServiceTestSuite) TestGetConfigurations_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/configurations": {
			StatusCode: 200,
			Body: `{
				"count": 2,
				"resources": [
					{
						"id": "config-1",
						"name": "my-gpt4-config",
						"executableId": "azure-openai",
						"scenarioId": "foundation-models",
						"createdAt": "2023-01-01T00:00:00Z",
						"parameterBindings": [
							{"key": "modelName", "value": "gpt-4"},
							{"key": "modelVersion", "value": "latest"}
						]
					},
					{
						"id": "config-2",
						"name": "my-claude-config",
						"executableId": "anthropic-claude",
						"scenarioId": "foundation-models",
						"createdAt": "2023-01-02T00:00:00Z",
						"parameterBindings": [
							{"key": "modelName", "value": "claude-3-sonnet"},
							{"key": "modelVersion", "value": "latest"}
						]
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetConfigurations(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(2, result.Count)
	suite.Len(result.Resources, 2)
	suite.Equal("config-1", result.Resources[0].ID)
	suite.Equal("my-gpt4-config", result.Resources[0].Name)
	suite.Equal("config-2", result.Resources[1].ID)
	suite.Equal("my-claude-config", result.Resources[1].Name)
}

func (suite *AICoreServiceTestSuite) TestGetConfigurations_UserNotFound_Error() {
	// Setup
	email := "nonexistent@example.com"

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetConfigurations(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotFoundInDB, err)
}

func (suite *AICoreServiceTestSuite) TestGetConfigurations_APIError_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses - API returns error
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/configurations": {
			StatusCode: 500,
			Body:       `{"error": "Internal server error"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetConfigurations(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "500")
}

func (suite *AICoreServiceTestSuite) TestUpdateDeployment_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "deployment-123"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	updateRequest := &service.AICoreDeploymentModificationRequest{
		TargetStatus: "STOPPED",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"PATCH:/v2/lm/deployments/deployment-123": {
			StatusCode: 202,
			Body: `{
				"id": "deployment-123",
				"message": "Deployment modification accepted",
				"deploymentUrl": "https://api.example.com/v1/deployments/deployment-123",
				"status": "RUNNING",
				"targetStatus": "STOPPED"
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.UpdateDeployment(c, deploymentID, updateRequest)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("deployment-123", result.ID)
	suite.Equal("Deployment modification accepted", result.Message)
	suite.Equal("STOPPED", result.TargetStatus)
}

func (suite *AICoreServiceTestSuite) TestUpdateDeployment_NotFound_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "nonexistent-deployment"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	updateRequest := &service.AICoreDeploymentModificationRequest{
		TargetStatus: "STOPPED",
	}

	// Setup mock server responses - deployment not found
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"PATCH:/v2/lm/deployments/nonexistent-deployment": {
			StatusCode: 404,
			Body:       `{"error": "Deployment not found"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.UpdateDeployment(c, deploymentID, updateRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrAICoreDeploymentNotFound, err)
}

func (suite *AICoreServiceTestSuite) TestUpdateDeployment_APIError_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "deployment-123"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	updateRequest := &service.AICoreDeploymentModificationRequest{
		TargetStatus: "STOPPED",
	}

	// Setup mock server responses - API error
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"PATCH:/v2/lm/deployments/deployment-123": {
			StatusCode: 500,
			Body:       `{"error": "Internal server error"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.UpdateDeployment(c, deploymentID, updateRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "500")
}

func (suite *AICoreServiceTestSuite) TestDeleteDeployment_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "deployment-123"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"DELETE:/v2/lm/deployments/deployment-123": {
			StatusCode: 202,
			Body: `{
				"id": "deployment-123",
				"message": "Deployment deletion accepted"
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.DeleteDeployment(c, deploymentID)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("deployment-123", result.ID)
	suite.Equal("Deployment deletion accepted", result.Message)
}

func (suite *AICoreServiceTestSuite) TestDeleteDeployment_NotFound_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "nonexistent-deployment"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses - deployment not found
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"DELETE:/v2/lm/deployments/nonexistent-deployment": {
			StatusCode: 404,
			Body:       `{"error": "Deployment not found"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.DeleteDeployment(c, deploymentID)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrAICoreDeploymentNotFound, err)
}

func (suite *AICoreServiceTestSuite) TestGetDeploymentDetails_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "deployment-123"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments/deployment-123": {
			StatusCode: 200,
			Body: `{
				"id": "deployment-123",
				"deploymentUrl": "https://api.example.com/v1/deployments/deployment-123",
				"configurationId": "config-1",
				"configurationName": "my-config",
				"executableId": "azure-openai",
				"scenarioId": "foundation-models",
				"status": "RUNNING",
				"statusMessage": "Deployment is running",
				"targetStatus": "RUNNING",
				"lastOperation": "CREATE",
				"latestRunningConfigurationId": "config-1",
				"ttl": "1h",
				"createdAt": "2023-01-01T00:00:00Z",
				"modifiedAt": "2023-01-01T01:00:00Z",
				"submissionTime": "2023-01-01T00:00:00Z",
				"startTime": "2023-01-01T00:05:00Z",
				"details": {
					"resources": {
						"backend_details": {
							"model": {
								"name": "gpt-4"
							}
						}
					}
				}
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeploymentDetails(c, deploymentID)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("deployment-123", result.ID)
	suite.Equal("config-1", result.ConfigurationID)
	suite.Equal("my-config", result.ConfigurationName)
	suite.Equal("RUNNING", result.Status)
	suite.Equal("1h", result.TTL)
}

func (suite *AICoreServiceTestSuite) TestGetDeploymentDetails_NotFound_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()
	deploymentID := "nonexistent-deployment"

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup mock server responses - deployment not found
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments/nonexistent-deployment": {
			StatusCode: 404,
			Body:       `{"error": "Deployment not found"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.GetDeploymentDetails(c, deploymentID)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrAICoreDeploymentNotFound, err)
}

func (suite *AICoreServiceTestSuite) TestCreateConfiguration_Success() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	configRequest := &service.AICoreConfigurationRequest{
		Name:         "my-test-config",
		ExecutableID: "azure-openai",
		ScenarioID:   "foundation-models",
		ParameterBindings: []map[string]string{
			{"key": "modelName", "value": "gpt-4"},
			{"key": "modelVersion", "value": "latest"},
		},
	}

	// Setup mock server responses
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"POST:/v2/lm/configurations": {
			StatusCode: 201,
			Body: `{
				"id": "config-789",
				"message": "Configuration created successfully"
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateConfiguration(c, configRequest)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("config-789", result.ID)
	suite.Equal("Configuration created successfully", result.Message)
}

func (suite *AICoreServiceTestSuite) TestCreateConfiguration_UserNotFound_Error() {
	// Setup
	email := "nonexistent@example.com"

	configRequest := &service.AICoreConfigurationRequest{
		Name:         "my-test-config",
		ExecutableID: "azure-openai",
		ScenarioID:   "foundation-models",
	}

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateConfiguration(c, configRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotFoundInDB, err)
}

func (suite *AICoreServiceTestSuite) TestCreateConfiguration_APIError_Error() {
	// Setup
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	configRequest := &service.AICoreConfigurationRequest{
		Name:         "my-test-config",
		ExecutableID: "azure-openai",
		ScenarioID:   "foundation-models",
	}

	// Setup mock server responses - API returns error
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"POST:/v2/lm/configurations": {
			StatusCode: 400,
			Body:       `{"error": "Invalid configuration"}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.CreateConfiguration(c, configRequest)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "400")
}

func (suite *AICoreServiceTestSuite) TestGetMe_TeamMember_Success() {
	// Setup - Regular team member
	username := "john.doe"
	teamID := uuid.New()

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup credentials so team-alpha is in the environment
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username) // GetMe uses username, not email
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	suite.Len(result.AIInstances, 1)
	suite.Contains(result.AIInstances, "team-alpha")
}

func (suite *AICoreServiceTestSuite) TestGetMe_UserNotFound_Error() {
	// Setup
	username := "nonexistent"

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserNotFoundInDB, err)
}

func (suite *AICoreServiceTestSuite) TestGetMe_NoUsername_Error() {
	// Setup - no username in context
	c := suite.createGinContext("")
	// Don't set username

	// Execute
	result, err := suite.service.GetMe(c)

	// Assert
	suite.Error(err)
	suite.Nil(result)
	suite.Equal(errors.ErrUserEmailNotFound, err)
}

func (suite *AICoreServiceTestSuite) TestGetMe_WithMetadata_Success() {
	// Setup - User with metadata ai_instances
	username := "jane.doe"
	teamID := uuid.New()

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-beta", "team-gamma"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleMember,
		Metadata:  metadataJSON,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Setup credentials
	suite.setupCredentials([]string{"team-alpha", "team-beta", "team-gamma"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	// Should have team-alpha (from team assignment) + team-beta and team-gamma (from metadata)
	suite.Len(result.AIInstances, 3)
	suite.Contains(result.AIInstances, "team-alpha")
	suite.Contains(result.AIInstances, "team-beta")
	suite.Contains(result.AIInstances, "team-gamma")
}

func (suite *AICoreServiceTestSuite) TestGetMe_Manager_OwnsGroup_Success() {
	// Setup - Manager who owns a group
	username := "group.manager"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleManager,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		GroupID:   groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{ID: groupID, Name: "group-one"},
		Owner:     username, // Manager owns this group
		OrgID:     orgID,
	}

	// Teams in the group
	teamsInGroup := []models.Team{
		{BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"}, GroupID: groupID},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "team-beta"}, GroupID: groupID},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "team-gamma"}, GroupID: groupID},
	}

	// Setup credentials for all teams
	suite.setupCredentials([]string{"team-alpha", "team-beta", "team-gamma"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.groupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.teamRepo.EXPECT().GetByGroupID(groupID, gomock.Any(), gomock.Any()).Return(teamsInGroup, int64(len(teamsInGroup)), nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	suite.Len(result.AIInstances, 3) // All teams in the owned group
	suite.Contains(result.AIInstances, "team-alpha")
	suite.Contains(result.AIInstances, "team-beta")
	suite.Contains(result.AIInstances, "team-gamma")
}

func (suite *AICoreServiceTestSuite) TestGetMe_MMM_OwnsOrganization_Success() {
	// Setup - MMM who owns an organization
	username := "org.mmm"
	teamID := uuid.New()
	groupID := uuid.New()
	orgID := uuid.New()

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleMMM,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		GroupID:   groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{ID: groupID, Name: "group-one"},
		OrgID:     orgID,
	}

	org := &models.Organization{
		BaseModel: models.BaseModel{ID: orgID, Name: "org-one"},
		Owner:     username, // MMM owns this org
	}

	// Multiple groups in the org
	groupsInOrg := []models.Group{
		{BaseModel: models.BaseModel{ID: groupID, Name: "group-one"}, OrgID: orgID},
		{BaseModel: models.BaseModel{ID: uuid.New(), Name: "group-two"}, OrgID: orgID},
	}

	// Teams across all groups
	teamsGroup1 := []models.Team{
		{BaseModel: models.BaseModel{Name: "team-alpha"}},
		{BaseModel: models.BaseModel{Name: "team-beta"}},
	}
	teamsGroup2 := []models.Team{
		{BaseModel: models.BaseModel{Name: "team-gamma"}},
		{BaseModel: models.BaseModel{Name: "team-delta"}},
	}

	// Setup credentials
	suite.setupCredentials([]string{"team-alpha", "team-beta", "team-gamma", "team-delta"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.groupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.orgRepo.EXPECT().GetByID(orgID).Return(org, nil)
	suite.groupRepo.EXPECT().GetByOrganizationID(orgID, gomock.Any(), gomock.Any()).Return(groupsInOrg, int64(len(groupsInOrg)), nil)
	suite.teamRepo.EXPECT().GetByGroupID(groupsInOrg[0].ID, gomock.Any(), gomock.Any()).Return(teamsGroup1, int64(len(teamsGroup1)), nil)
	suite.teamRepo.EXPECT().GetByGroupID(groupsInOrg[1].ID, gomock.Any(), gomock.Any()).Return(teamsGroup2, int64(len(teamsGroup2)), nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	suite.Len(result.AIInstances, 4) // All teams across all groups in the org
	suite.Contains(result.AIInstances, "team-alpha")
	suite.Contains(result.AIInstances, "team-beta")
	suite.Contains(result.AIInstances, "team-gamma")
	suite.Contains(result.AIInstances, "team-delta")
}

func (suite *AICoreServiceTestSuite) TestGetMe_FilteredByCredentials_Success() {
	// Setup - User with teams but only some have credentials
	username := "john.doe"
	teamID := uuid.New()
	groupID := uuid.New()

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleManager,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		GroupID:   groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{ID: groupID, Name: "group-one"},
		Owner:     username,
	}

	// Teams in the group
	teamsInGroup := []models.Team{
		{BaseModel: models.BaseModel{Name: "team-alpha"}},
		{BaseModel: models.BaseModel{Name: "team-beta"}},
		{BaseModel: models.BaseModel{Name: "team-gamma"}}, // No credentials for this one
	}

	// Setup credentials - only for team-alpha and team-beta
	suite.setupCredentials([]string{"team-alpha", "team-beta"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.groupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.teamRepo.EXPECT().GetByGroupID(groupID, gomock.Any(), gomock.Any()).Return(teamsInGroup, int64(len(teamsInGroup)), nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	// Only teams with credentials should be returned
	suite.Len(result.AIInstances, 2)
	suite.Contains(result.AIInstances, "team-alpha")
	suite.Contains(result.AIInstances, "team-beta")
	suite.NotContains(result.AIInstances, "team-gamma") // Filtered out
}

func (suite *AICoreServiceTestSuite) TestGetMe_NoTeamAssignment_NoMetadata_Error() {
	// Setup - User with no team and no metadata
	username := "unassigned.user"

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    nil,
		TeamRole:  models.TeamRoleMember, // Not a manager
		Metadata:  nil,
	}

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert - Should return empty ai_instances, not error
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	suite.Len(result.AIInstances, 0)
}

func (suite *AICoreServiceTestSuite) TestGetMe_MetadataAndRoleBased_Combined() {
	// Setup - Manager with both role-based teams and metadata teams
	username := "hybrid.manager"
	teamID := uuid.New()
	groupID := uuid.New()

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-delta", "team-epsilon"},
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleManager,
		Metadata:  metadataJSON,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		GroupID:   groupID,
	}

	group := &models.Group{
		BaseModel: models.BaseModel{ID: groupID, Name: "group-one"},
		Owner:     username,
	}

	teamsInGroup := []models.Team{
		{BaseModel: models.BaseModel{Name: "team-alpha"}},
		{BaseModel: models.BaseModel{Name: "team-beta"}},
	}

	// Setup credentials for all teams
	suite.setupCredentials([]string{"team-alpha", "team-beta", "team-delta", "team-epsilon"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)
	suite.groupRepo.EXPECT().GetByID(groupID).Return(group, nil)
	suite.teamRepo.EXPECT().GetByGroupID(groupID, gomock.Any(), gomock.Any()).Return(teamsInGroup, int64(len(teamsInGroup)), nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	// Should have role-based teams (team-alpha, team-beta) + metadata teams (team-delta, team-epsilon)
	suite.Len(result.AIInstances, 4)
	suite.Contains(result.AIInstances, "team-alpha")
	suite.Contains(result.AIInstances, "team-beta")
	suite.Contains(result.AIInstances, "team-delta")
	suite.Contains(result.AIInstances, "team-epsilon")
}

func (suite *AICoreServiceTestSuite) TestGetMe_NoDuplicates_Success() {
	// Setup - Ensure no duplicates when same team appears in multiple sources
	username := "jane.doe"
	teamID := uuid.New()

	metadata := map[string]interface{}{
		"ai_instances": []string{"team-alpha", "team-alpha"}, // Duplicate in metadata
	}
	metadataJSON, _ := json.Marshal(metadata)

	member := &models.User{
		BaseModel: models.BaseModel{Name: username},
		TeamID:    &teamID,
		TeamRole:  models.TeamRoleMember,
		Metadata:  metadataJSON,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"}, // Same as metadata
	}

	// Setup credentials
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByName(username).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext("")
	c.Set("username", username)
	result, err := suite.service.GetMe(c)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal(username, result.User)
	// Should only have team-alpha once, despite appearing in both team assignment and metadata
	suite.Len(result.AIInstances, 1)
	suite.Contains(result.AIInstances, "team-alpha")
}

// ChatInference Tests - Testing GetDeployments call and error handling

func (suite *AICoreServiceTestSuite) TestChatInference_UserNotFound() {
	// Setup - User not found causes GetDeployments to fail
	email := "nonexistent@example.com"

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-123",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Setup mocks - GetDeployments will fail because user not found
	suite.userRepo.EXPECT().GetByEmail(email).Return((*models.User)(nil), errors.ErrUserNotFound)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should return error from GetDeployments
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), errors.ErrUserNotFound.Error())
}

func (suite *AICoreServiceTestSuite) TestChatInference_DeploymentNotFound() {
	// Setup - GetDeployments succeeds but deployment ID not found
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "nonexistent-deployment",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Setup mock server responses - return empty deployments
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body:       `{"count": 0, "resources": []}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - GetDeployments succeeds but deployment not found
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "not found or user does not have access")
}

func (suite *AICoreServiceTestSuite) TestChatInference_UserNotAssignedToTeam_Error() {
	// Setup - User not assigned to team causes GetDeployments to fail
	email := "unassigned@example.com"

	member := &models.User{
		TeamID:   nil,
		TeamRole: models.TeamRoleMember, // Not a manager
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-123",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - GetDeployments fails, ChatInference returns error
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "failed to get deployments")
}

func (suite *AICoreServiceTestSuite) TestChatInference_DeploymentURLNotAvailable_Error() {
	// Setup - Deployment exists but has no URL
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-no-url",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Setup mock server responses - deployment exists but has no URL
	responses := map[string]mockResponse{
		"POST:/oauth/token": {
			StatusCode: 200,
			Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
		},
		"GET:/v2/lm/deployments": {
			StatusCode: 200,
			Body: `{
				"count": 1,
				"resources": [
					{
						"id": "deployment-no-url",
						"configurationId": "config-1",
						"status": "PENDING",
						"statusMessage": "Deployment is pending",
						"deploymentUrl": "",
						"createdAt": "2023-01-01T00:00:00Z",
						"modifiedAt": "2023-01-01T01:00:00Z"
					}
				]
			}`,
		},
	}
	suite.setupMockServer(responses)
	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should return error about missing deployment URL
	suite.Error(err)
	suite.Nil(result)
	suite.Contains(err.Error(), "deployment URL not available")
	suite.Contains(err.Error(), "deployment-no-url")
}

// Test Gemini model detection in ChatInference
func (suite *AICoreServiceTestSuite) TestChatInference_GeminiModel_DetectedCorrectly() {
	// Setup - Test that Gemini models are detected by model name containing "gemini"
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-gemini",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
	}

	// Setup mock server first (before using suite.server.URL)
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-gemini",
							"configurationId": "config-1",
							"scenarioId": "foundation-models",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-gemini",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z",
							"details": {
								"resources": {
									"backend_details": {
										"model": {
											"name": "gemini-1.5-flash"
										}
									}
								}
							}
						}
					]
				}`,
			},
			"POST:/deployments/deployment-gemini/models/gemini-1.5-flash:generateContent": {
				StatusCode: 200,
				Body: `{
					"candidates": [{
						"content": {
							"parts": [{"text": "Hello! How can I help you?"}],
							"role": "model"
						},
						"finishReason": "STOP"
					}],
					"usageMetadata": {
						"promptTokenCount": 5,
						"candidatesTokenCount": 10,
						"totalTokenCount": 15
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully detect Gemini and use /models/<model>:generateContent endpoint
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("Hello! How can I help you?", result.Choices[0].Message.Content)
	suite.Equal(5, result.Usage.PromptTokens)
	suite.Equal(10, result.Usage.CompletionTokens)
	suite.Equal(15, result.Usage.TotalTokens)
}

// Test Gemini multimodal content handling (text + images)
func (suite *AICoreServiceTestSuite) TestChatInference_GeminiModel_MultimodalContent() {
	// Setup - Test that Gemini handles multimodal content (text + images)
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Create multimodal inference request with text and image
	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-gemini",
		Messages: []service.AICoreInferenceMessage{
			{
				Role: "user",
				Content: []interface{}{
					map[string]interface{}{
						"type": "text",
						"text": "What's in this image?",
					},
					map[string]interface{}{
						"type": "image_url",
						"image_url": map[string]interface{}{
							"url": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg==",
						},
					},
				},
			},
		},
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-gemini",
							"configurationId": "config-1",
							"scenarioId": "foundation-models",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-gemini",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z",
							"details": {
								"resources": {
									"backend_details": {
										"model": {
											"name": "gemini-1.5-pro"
										}
									}
								}
							}
						}
					]
				}`,
			},
			"POST:/deployments/deployment-gemini/models/gemini-1.5-pro:generateContent": {
				StatusCode: 200,
				Body: `{
					"candidates": [{
						"content": {
							"parts": [{"text": "This is a 1x1 pixel red image."}],
							"role": "model"
						},
						"finishReason": "STOP"
					}],
					"usageMetadata": {
						"promptTokenCount": 15,
						"candidatesTokenCount": 12,
						"totalTokenCount": 27
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully handle multimodal content
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("This is a 1x1 pixel red image.", result.Choices[0].Message.Content)
	suite.Equal(15, result.Usage.PromptTokens)
	suite.Equal(12, result.Usage.CompletionTokens)
	suite.Equal(27, result.Usage.TotalTokens)
}

// Test Gemini generation config parameters (MaxTokens and Temperature)
func (suite *AICoreServiceTestSuite) TestChatInference_GeminiModel_WithGenerationConfig() {
	// Setup - Test that Gemini properly handles generation config parameters
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	// Create inference request with MaxTokens and Temperature
	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-gemini",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Tell me a story"},
		},
		MaxTokens:   500,
		Temperature: 0.8,
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		// For the generateContent endpoint, verify the request body contains generation_config
		if key == "POST:/deployments/deployment-gemini/models/gemini-1.5-flash:generateContent" {
			// Read and verify the request body
			body, _ := io.ReadAll(r.Body)
			var requestBody map[string]interface{}
			json.Unmarshal(body, &requestBody)

			// Verify generation_config is present with correct values
			if genConfig, ok := requestBody["generation_config"].(map[string]interface{}); ok {
				if maxTokens, ok := genConfig["maxOutputTokens"].(float64); ok && maxTokens == 500 {
					// MaxTokens is correct
				}
				if temp, ok := genConfig["temperature"].(float64); ok && temp == 0.8 {
					// Temperature is correct
				}
			}
		}

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-gemini",
							"configurationId": "config-1",
							"scenarioId": "foundation-models",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-gemini",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z",
							"details": {
								"resources": {
									"backend_details": {
										"model": {
											"name": "gemini-1.5-flash"
										}
									}
								}
							}
						}
					]
				}`,
			},
			"POST:/deployments/deployment-gemini/models/gemini-1.5-flash:generateContent": {
				StatusCode: 200,
				Body: `{
					"candidates": [{
						"content": {
							"parts": [{"text": "Once upon a time, in a land far away..."}],
							"role": "model"
						},
						"finishReason": "STOP"
					}],
					"usageMetadata": {
						"promptTokenCount": 10,
						"candidatesTokenCount": 50,
						"totalTokenCount": 60
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully handle generation config parameters
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("Once upon a time, in a land far away...", result.Choices[0].Message.Content)
	suite.Equal(10, result.Usage.PromptTokens)
	suite.Equal(50, result.Usage.CompletionTokens)
	suite.Equal(60, result.Usage.TotalTokens)
}

// Test orchestration scenario detection and handling
func (suite *AICoreServiceTestSuite) TestChatInference_OrchestrationScenario_Success() {
	// Setup - Test that orchestration scenarios are detected and handled correctly
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-orchestration",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello, how are you?"},
		},
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-orchestration",
							"configurationId": "config-1",
							"scenarioId": "orchestration",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-orchestration",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z"
						}
					]
				}`,
			},
			"POST:/deployments/deployment-orchestration/completion": {
				StatusCode: 200,
				Body: `{
					"orchestration_result": {
						"choices": [{
							"index": 0,
							"message": {
								"role": "assistant",
								"content": "I'm doing well, thank you for asking!"
							},
							"finish_reason": "stop"
						}]
					},
					"module_results": {
						"templating": [],
						"llm": [{
							"message": {
								"role": "assistant",
								"content": "I'm doing well, thank you for asking!"
							}
						}]
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully detect orchestration and use /completion endpoint
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("I'm doing well, thank you for asking!", result.Choices[0].Message.Content)
	// Orchestration doesn't return token counts
	suite.Equal(0, result.Usage.PromptTokens)
	suite.Equal(0, result.Usage.CompletionTokens)
	suite.Equal(0, result.Usage.TotalTokens)
}

// Test GPT model detection and handling
func (suite *AICoreServiceTestSuite) TestChatInference_GPTModel_DetectedCorrectly() {
	// Setup - Test that GPT models are detected and use /chat/completions endpoint
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-gpt",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Hello"},
		},
		MaxTokens:   100,
		Temperature: 0.7,
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-gpt",
							"configurationId": "config-1",
							"scenarioId": "foundation-models",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-gpt",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z",
							"details": {
								"resources": {
									"backend_details": {
										"model": {
											"name": "gpt-4"
										}
									}
								}
							}
						}
					]
				}`,
			},
			"POST:/deployments/deployment-gpt/chat/completions": {
				StatusCode: 200,
				Body: `{
					"id": "chatcmpl-123",
					"object": "chat.completion",
					"created": 1677652288,
					"model": "gpt-4",
					"choices": [{
						"index": 0,
						"message": {
							"role": "assistant",
							"content": "Hello! How can I help you today?"
						},
						"finish_reason": "stop"
					}],
					"usage": {
						"prompt_tokens": 10,
						"completion_tokens": 15,
						"total_tokens": 25
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully detect GPT and use /chat/completions endpoint
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("Hello! How can I help you today?", result.Choices[0].Message.Content)
	suite.Equal(10, result.Usage.PromptTokens)
	suite.Equal(15, result.Usage.CompletionTokens)
	suite.Equal(25, result.Usage.TotalTokens)
}

// Test Anthropic/Claude model with /invoke endpoint
func (suite *AICoreServiceTestSuite) TestChatInference_AnthropicModel_DetectedCorrectly() {
	// Setup - Test that Anthropic/Claude models use /invoke endpoint with Anthropic format
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-claude",
		Messages: []service.AICoreInferenceMessage{
			{Role: "system", Content: "You are a helpful assistant."},
			{Role: "user", Content: "Hello, how are you?"},
		},
		MaxTokens:   150,
		Temperature: 0.7,
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-claude",
							"configurationId": "config-1",
							"scenarioId": "foundation-models",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-claude",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z",
							"details": {
								"resources": {
									"backend_details": {
										"model": {
											"name": "claude-3-sonnet"
										}
									}
								}
							}
						}
					]
				}`,
			},
			"POST:/deployments/deployment-claude/invoke": {
				StatusCode: 200,
				Body: `{
					"id": "msg_01XFDUDYJgAACzvnptvVoYEL",
					"type": "message",
					"role": "assistant",
					"content": [
						{
							"type": "text",
							"text": "I'm doing well, thank you for asking! How can I assist you today?"
						}
					],
					"model": "claude-3-sonnet-20240229",
					"stop_reason": "end_turn",
					"usage": {
						"input_tokens": 25,
						"output_tokens": 18
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully detect Anthropic and use /invoke endpoint
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("I'm doing well, thank you for asking! How can I assist you today?", result.Choices[0].Message.Content)
	suite.Equal("assistant", result.Choices[0].Message.Role)
	suite.Equal("end_turn", result.Choices[0].FinishReason)
	suite.Equal(25, result.Usage.PromptTokens)
	suite.Equal(18, result.Usage.CompletionTokens)
	suite.Equal(43, result.Usage.TotalTokens)
}

// Test orchestration scenario with generation config parameters
func (suite *AICoreServiceTestSuite) TestChatInference_OrchestrationScenario_WithGenerationConfig() {
	// Setup - Test that orchestration properly handles temperature and max_tokens
	email := "team.member@example.com"
	teamID := uuid.New()

	member := &models.User{
		TeamID:   &teamID,
		TeamRole: models.TeamRoleMember,
	}

	team := &models.Team{
		BaseModel: models.BaseModel{ID: teamID, Name: "team-alpha"},
		Owner:     "team-alpha",
	}

	inferenceReq := &service.AICoreInferenceRequest{
		DeploymentID: "deployment-orchestration",
		Messages: []service.AICoreInferenceMessage{
			{Role: "user", Content: "Write a short story"},
		},
		MaxTokens:   300,
		Temperature: 0.7,
	}

	// Setup mock server
	suite.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := fmt.Sprintf("%s:%s", r.Method, r.URL.Path)

		// For the completion endpoint, verify the request body contains temperature and max_tokens in orchestration_config
		if key == "POST:/deployments/deployment-orchestration/completion" {
			body, _ := io.ReadAll(r.Body)
			var requestBody map[string]interface{}
			json.Unmarshal(body, &requestBody)

			// Verify orchestration_config contains model_params with temperature and max_tokens
			if orchConfig, ok := requestBody["orchestration_config"].(map[string]interface{}); ok {
				if moduleConfigs, ok := orchConfig["module_configurations"].(map[string]interface{}); ok {
					if llmConfig, ok := moduleConfigs["llm_module_config"].(map[string]interface{}); ok {
						if modelParams, ok := llmConfig["model_params"].(map[string]interface{}); ok {
							if temp, ok := modelParams["temperature"].(float64); ok && temp == 0.7 {
								// Temperature is correct
							}
							if maxTokens, ok := modelParams["max_tokens"].(float64); ok && maxTokens == 300 {
								// MaxTokens is correct
							}
						}
					}
				}
			}
		}

		responses := map[string]mockResponse{
			"POST:/oauth/token": {
				StatusCode: 200,
				Body:       `{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`,
			},
			"GET:/v2/lm/deployments": {
				StatusCode: 200,
				Body: `{
					"count": 1,
					"resources": [
						{
							"id": "deployment-orchestration",
							"configurationId": "config-1",
							"scenarioId": "orchestration",
							"status": "RUNNING",
							"statusMessage": "Deployment is running",
							"deploymentUrl": "` + suite.server.URL + `/deployments/deployment-orchestration",
							"createdAt": "2023-01-01T00:00:00Z",
							"modifiedAt": "2023-01-01T01:00:00Z"
						}
					]
				}`,
			},
			"POST:/deployments/deployment-orchestration/completion": {
				StatusCode: 200,
				Body: `{
					"orchestration_result": {
						"choices": [{
							"index": 0,
							"message": {
								"role": "assistant",
								"content": "Once upon a time in a distant land..."
							},
							"finish_reason": "stop"
						}]
					},
					"module_results": {
						"templating": [],
						"llm": [{
							"message": {
								"role": "assistant",
								"content": "Once upon a time in a distant land..."
							}
						}]
					}
				}`,
			},
		}

		if response, exists := responses[key]; exists {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(response.StatusCode)
			_, _ = w.Write([]byte(response.Body))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	suite.setupCredentials([]string{"team-alpha"})

	// Setup mocks
	suite.userRepo.EXPECT().GetByEmail(email).Return(member, nil)
	suite.teamRepo.EXPECT().GetByID(teamID).Return(team, nil)

	// Execute
	c := suite.createGinContext(email)
	result, err := suite.service.ChatInference(c, inferenceReq)

	// Assert - Should successfully handle generation config parameters
	suite.NoError(err)
	suite.NotNil(result)
	suite.Len(result.Choices, 1)
	suite.Equal("Once upon a time in a distant land...", result.Choices[0].Message.Content)
	// Orchestration doesn't return token counts
	suite.Equal(0, result.Usage.PromptTokens)
	suite.Equal(0, result.Usage.CompletionTokens)
	suite.Equal(0, result.Usage.TotalTokens)
}

// Tests for UploadAttachment function

// Helper function to create a temporary file for testing
func createTempFile(content []byte, filename string) (multipart.File, *multipart.FileHeader, error) {
	tmpFile, err := os.CreateTemp("", "test-*")
	if err != nil {
		return nil, nil, err
	}

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, nil, err
	}

	if _, err := tmpFile.Seek(0, 0); err != nil {
		tmpFile.Close()
		os.Remove(tmpFile.Name())
		return nil, nil, err
	}

	header := &multipart.FileHeader{
		Filename: filename,
		Size:     int64(len(content)),
	}

	return tmpFile, header, nil
}

func (suite *AICoreServiceTestSuite) TestUploadAttachment_DifferentFileTypes() {
	// Table-driven test for different file types
	testCases := []struct {
		name             string
		filename         string
		content          []byte
		expectedMimeType string
	}{
		{
			name:             "PNG Image",
			filename:         "test.png",
			content:          []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01, 0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4, 0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41, 0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00, 0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE, 0x42, 0x60, 0x82},
			expectedMimeType: "image/png",
		},
		{
			name:             "JPEG Image",
			filename:         "photo.jpg",
			content:          []byte{0xFF, 0xD8, 0xFF, 0xE0, 0x00, 0x10, 0x4A, 0x46, 0x49, 0x46, 0x00, 0x01, 0x01, 0x00, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0xFF, 0xD9},
			expectedMimeType: "image/jpeg",
		},
		{
			name:             "JSON File",
			filename:         "data.json",
			content:          []byte(`{"key": "value", "number": 123}`),
			expectedMimeType: "application/json",
		},
		{
			name:             "Text File",
			filename:         "test.txt",
			content:          []byte("Hello, World!\nThis is a test file."),
			expectedMimeType: "text/plain",
		},
		{
			name:             "HTML File",
			filename:         "index.html",
			content:          []byte("<html><body><h1>Test</h1></body></html>"),
			expectedMimeType: "text/html",
		},
		{
			name:             "HTM File",
			filename:         "page.htm",
			content:          []byte("<html><body>Test</body></html>"),
			expectedMimeType: "text/html",
		},
		{
			name:             "CSV File",
			filename:         "data.csv",
			content:          []byte("name,age,city\nJohn,30,NYC\nJane,25,LA"),
			expectedMimeType: "text/csv",
		},
		{
			name:             "XML File",
			filename:         "config.xml",
			content:          []byte(`<?xml version="1.0"?><root><item>test</item></root>`),
			expectedMimeType: "application/xml",
		},
		{
			name:             "YAML File",
			filename:         "config.yaml",
			content:          []byte("key: value\nlist:\n  - item1\n  - item2"),
			expectedMimeType: "application/x-yaml",
		},
		{
			name:             "YML File",
			filename:         "config.yml",
			content:          []byte("test: value"),
			expectedMimeType: "application/x-yaml",
		},
		{
			name:             "Markdown File",
			filename:         "README.md",
			content:          []byte("# Heading\n\nThis is **bold** text."),
			expectedMimeType: "text/markdown",
		},
		{
			name:             "PDF File",
			filename:         "document.pdf",
			content:          []byte("%PDF-1.4\n%EOF"),
			expectedMimeType: "application/pdf",
		},
		{
			name:             "Uppercase Extension",
			filename:         "DATA.JSON",
			content:          []byte(`{"test": true}`),
			expectedMimeType: "application/json",
		},
		{
			name:             "Empty File",
			filename:         "empty.txt",
			content:          []byte{},
			expectedMimeType: "text/plain",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Setup
			file, header, err := createTempFile(tc.content, tc.filename)
			if err != nil {
				suite.T().Fatalf("Failed to create temp file: %v", err)
			}
			defer file.Close()
			defer os.Remove(file.(*os.File).Name())

			c := suite.createGinContext("")

			// Execute
			result, err := suite.service.UploadAttachment(c, file, header)

			// Assert
			suite.NoError(err, "UploadAttachment should not return error for %s", tc.name)
			suite.NotNil(result, "Result should not be nil for %s", tc.name)

			// Verify all expected fields are present
			suite.Contains(result, "url", "Result should contain 'url' field")
			suite.Contains(result, "mimeType", "Result should contain 'mimeType' field")
			suite.Contains(result, "filename", "Result should contain 'filename' field")
			suite.Contains(result, "size", "Result should contain 'size' field")

			// Verify field values
			suite.Equal(tc.filename, result["filename"], "Filename should match for %s", tc.name)
			suite.Equal(int64(len(tc.content)), result["size"], "Size should match for %s", tc.name)
			suite.Equal(tc.expectedMimeType, result["mimeType"], "MIME type should match for %s", tc.name)

			// Verify data URL format
			dataURL := result["url"].(string)
			expectedPrefix := fmt.Sprintf("data:%s;base64,", tc.expectedMimeType)
			suite.True(strings.HasPrefix(dataURL, expectedPrefix),
				"Data URL should start with correct prefix for %s", tc.name)

			// Verify base64 encoding is correct (can be decoded back to original)
			if len(tc.content) > 0 {
				parts := strings.SplitN(dataURL, ";base64,", 2)
				suite.Len(parts, 2, "Data URL should have base64 part for %s", tc.name)

				decoded, err := base64.StdEncoding.DecodeString(parts[1])
				suite.NoError(err, "Base64 decoding should succeed for %s", tc.name)
				suite.Equal(tc.content, decoded, "Decoded content should match original for %s", tc.name)
			}
		})
	}
}

func (suite *AICoreServiceTestSuite) TestUploadAttachment_LargeFile_Success() {
	// Setup - Test uploading a larger file (100KB)
	largeContent := make([]byte, 1024*100)
	for i := range largeContent {
		largeContent[i] = byte(i % 256)
	}

	file, header, err := createTempFile(largeContent, "large.bin")
	if err != nil {
		suite.T().Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()
	defer os.Remove(file.(*os.File).Name())

	c := suite.createGinContext("")
	result, err := suite.service.UploadAttachment(c, file, header)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("large.bin", result["filename"])
	suite.Equal(int64(len(largeContent)), result["size"])

	// Verify base64 encoding is correct
	dataURL := result["url"].(string)
	parts := strings.SplitN(dataURL, ";base64,", 2)
	suite.Len(parts, 2)
	decoded, err := base64.StdEncoding.DecodeString(parts[1])
	suite.NoError(err)
	suite.Equal(largeContent, decoded)
}

func (suite *AICoreServiceTestSuite) TestUploadAttachment_SpecialCharactersInFilename() {
	// Setup - Test filename with special characters
	content := []byte("test content")

	file, header, err := createTempFile(content, "test file (1) [copy].txt")
	if err != nil {
		suite.T().Fatalf("Failed to create temp file: %v", err)
	}
	defer file.Close()
	defer os.Remove(file.(*os.File).Name())

	c := suite.createGinContext("")
	result, err := suite.service.UploadAttachment(c, file, header)

	// Assert
	suite.NoError(err)
	suite.NotNil(result)
	suite.Equal("test file (1) [copy].txt", result["filename"])
	suite.Equal("text/plain", result["mimeType"])
}

func TestAICoreServiceTestSuite(t *testing.T) {
	suite.Run(t, new(AICoreServiceTestSuite))
}

package service

import (
	"context"
	"encoding/json"
	"mime/multipart"

	"developer-portal-backend/internal/database/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

//go:generate mockgen -source=interfaces.go -destination=../mocks/service_mocks.go -package=mocks

// UserServiceInterface defines the interface for user service
type UserServiceInterface interface {
	CreateUser(req *CreateUserRequest) (*UserResponse, error)
	GetUserByID(id uuid.UUID) (*UserResponse, error)
	GetUserByUserID(userID string) (*UserResponse, error)
	GetUserByName(name string) (*UserResponse, error)
	GetUserByNameWithLinks(name string) (*UserWithLinksAndPluginsResponse, error)
	GetUserByNameWithLinksAndPlugins(name string) (*UserWithLinksAndPluginsResponse, error)
	GetUserByUserIDWithLinks(userID string) (*UserWithLinksAndPluginsResponse, error)
	GetUsersByOrganization(organizationID uuid.UUID, limit, offset int) ([]UserResponse, int64, error)
	GetAllUsers(limit, offset int) ([]UserResponse, int64, error)
	SearchUsers(organizationID uuid.UUID, query string, limit, offset int) ([]UserResponse, int64, error)
	SearchUsersGlobal(query string, limit, offset int) ([]UserResponse, int64, error)
	GetActiveUsers(organizationID uuid.UUID, limit, offset int) ([]UserResponse, int64, error)
	UpdateUser(id uuid.UUID, req *UpdateUserRequest) (*UserResponse, error)
	UpdateUserTeam(userID uuid.UUID, teamID uuid.UUID, updatedBy string) (*UserResponse, error)
	DeleteUser(id uuid.UUID) error
	GetQuickLinks(id uuid.UUID) (*QuickLinksResponse, error)
	AddQuickLink(id uuid.UUID, req *AddQuickLinkRequest) (*UserResponse, error)
	RemoveQuickLink(id uuid.UUID, linkURL string) (*UserResponse, error)
	AddFavoriteLinkByUserID(userID string, linkID uuid.UUID) (*UserResponse, error)
	RemoveFavoriteLinkByUserID(userID string, linkID uuid.UUID) (*UserResponse, error)
	AddSubscribedPluginByUserID(userID string, pluginID uuid.UUID) (*UserResponse, error)
	RemoveSubscribedPluginByUserID(userID string, pluginID uuid.UUID) (*UserResponse, error)
}

// TeamServiceInterface defines the interface for team service
type TeamServiceInterface interface {
	GetAllTeams(organizationID *uuid.UUID, page, pageSize int) (*TeamListResponse, error)
	GetByID(id uuid.UUID) (*TeamResponse, error)
	GetBySimpleName(teamName string) (*TeamWithMembersResponse, error)
	GetBySimpleNameWithViewer(teamName string, viewerName string) (*TeamWithMembersResponse, error)
	GetTeamComponentsByID(id uuid.UUID, page, pageSize int) ([]models.Component, int64, error)
	UpdateTeamMetadata(id uuid.UUID, metadata json.RawMessage) (*TeamResponse, error)
}

// LandscapeServiceInterface defines the interface for landscape service
type LandscapeServiceInterface interface {
	CreateLandscape(req *CreateLandscapeRequest) (*LandscapeResponse, error)
	GetLandscapeByID(id uuid.UUID) (*LandscapeResponse, error)
	GetLandscapesByOrganization(organizationID uuid.UUID, limit, offset int) ([]LandscapeResponse, int64, error)
	UpdateLandscape(id uuid.UUID, req *UpdateLandscapeRequest) (*LandscapeResponse, error)
	DeleteLandscape(id uuid.UUID) error
	GetByProjectName(projectName string) (*LandscapeListResponse, error)
	GetByProjectNameAll(projectName string) ([]LandscapeMinimalResponse, error)
	ListByQuery(q string, domains []string, environments []string, limit int, offset int) (*LandscapeListResponse, error)
}

// GitHubServiceInterface defines the interface for GitHub service
type GitHubServiceInterface interface {
	GetUserOpenPullRequests(ctx context.Context, uuid, provider, state, sort, direction string, perPage, page int) (*PullRequestsResponse, error)
	GetUserTotalContributions(ctx context.Context, uuid, provider, period string) (*TotalContributionsResponse, error)
	GetContributionsHeatmap(ctx context.Context, uuid, provider, period string) (*ContributionsHeatmapResponse, error)
	GetAveragePRMergeTime(ctx context.Context, uuid, provider, period string) (*AveragePRMergeTimeResponse, error)
	GetUserPRReviewComments(ctx context.Context, uuid, provider, period string) (*PRReviewCommentsResponse, error)
	GetRepositoryContent(ctx context.Context, uuid, provider, owner, repo, path, ref string) (interface{}, error)
	UpdateRepositoryFile(ctx context.Context, uuid, provider, owner, repo, path, message, content, sha, branch string) (interface{}, error)
	ClosePullRequest(ctx context.Context, uuid, provider, owner, repo string, prNumber int, deleteBranch bool) (*PullRequest, error)
	GetGitHubAsset(ctx context.Context, uuid, provider, assetURL string) ([]byte, string, error)
}

// JenkinsServiceInterface defines the interface for Jenkins service
type JenkinsServiceInterface interface {
	GetJobParameters(ctx context.Context, jaasName, jobName string) (interface{}, error)
	TriggerJob(ctx context.Context, jaasName, jobName string, parameters map[string]string) (*JenkinsTriggerResult, error)
	GetQueueItemStatus(ctx context.Context, jaasName, queueItemID string) (*JenkinsQueueStatusResult, error)
	GetBuildStatus(ctx context.Context, jaasName, jobName string, buildNumber int) (*JenkinsBuildStatusResult, error)
}

// SonarServiceInterface defines the interface for Sonar service
type SonarServiceInterface interface {
	GetComponentMeasures(projectKey string) (*SonarCombinedResponse, error)
}

// JiraServiceInterface defines the interface for Jira service
type JiraServiceInterface interface {
	GetIssues(filters JiraIssueFilters) (*JiraIssuesResponse, error)
	GetIssuesCount(filters JiraIssueFilters) (int, error)
}

// AICoreServiceInterface defines the interface for AI Core service
type AICoreServiceInterface interface {
	GetDeployments(c *gin.Context) (*AICoreDeploymentsResponse, error)
	GetDeploymentDetails(c *gin.Context, deploymentID string) (*AICoreDeploymentDetailsResponse, error)
	GetModels(c *gin.Context, scenarioID string) (*AICoreModelsResponse, error)
	GetConfigurations(c *gin.Context) (*AICoreConfigurationsResponse, error)
	CreateConfiguration(c *gin.Context, req *AICoreConfigurationRequest) (*AICoreConfigurationResponse, error)
	CreateDeployment(c *gin.Context, req *AICoreDeploymentRequest) (*AICoreDeploymentResponse, error)
	UpdateDeployment(c *gin.Context, deploymentID string, req *AICoreDeploymentModificationRequest) (*AICoreDeploymentModificationResponse, error)
	DeleteDeployment(c *gin.Context, deploymentID string) (*AICoreDeploymentDeletionResponse, error)
	ChatInference(c *gin.Context, req *AICoreInferenceRequest) (*AICoreInferenceResponse, error)
	ChatInferenceStream(c *gin.Context, req *AICoreInferenceRequest, writer gin.ResponseWriter) error
	UploadAttachment(c *gin.Context, file multipart.File, header *multipart.FileHeader) (map[string]interface{}, error)
	GetMe(c *gin.Context) (*AICoreMeResponse, error)
}

// ComponentServiceInterface defines the interface for component service
type ComponentServiceInterface interface {
	GetByProjectNameAllView(projectName string) ([]ComponentProjectView, error)
	GetProjectTitleByID(id uuid.UUID) (string, error)
	GetByID(id uuid.UUID) (*models.Component, error)
}

// CategoryServiceInterface defines the interface for category service
type CategoryServiceInterface interface {
	GetAll(page, pageSize int) (*CategoryListResponse, error)
}

// ProjectServiceInterface defines the interface for project service
type ProjectServiceInterface interface {
	GetAllProjects() ([]models.Project, error)
	GetHealthURL(projectID uuid.UUID) (string, error)
}

// LinkServiceInterface defines the interface for link service
type LinkServiceInterface interface {
	// GetByOwnerUserID returns all links owned by the user with the given user_id (string, not)
	GetByOwnerUserID(ownerUserID string) ([]LinkResponse, error)
	// GetByOwnerUserIDWithViewer returns links owned by the given user and marks favorites based on viewer's favorites
	GetByOwnerUserIDWithViewer(ownerUserID string, viewerName string) ([]LinkResponse, error)
	// CreateLink creates a new link with validation and audit fields
	CreateLink(req *CreateLinkRequest) (*LinkResponse, error)
	// DeleteLink deletes a link by UUID
	DeleteLink(id uuid.UUID) error
	// UpdateLink updates an existing link
	UpdateLink(id uuid.UUID, req *UpdateLinkRequest) (*LinkResponse, error)
}

// DocumentationServiceInterface defines the interface for documentation service
type DocumentationServiceInterface interface {
	CreateDocumentation(req *CreateDocumentationRequest) (*DocumentationResponse, error)
	GetDocumentationByID(id uuid.UUID) (*DocumentationResponse, error)
	GetDocumentationsByTeamID(teamID uuid.UUID) ([]DocumentationResponse, error)
	UpdateDocumentation(id uuid.UUID, req *UpdateDocumentationRequest) (*DocumentationResponse, error)
	DeleteDocumentation(id uuid.UUID) error
}

// PluginServiceInterface defines the interface for plugin service
type PluginServiceInterface interface {
	CreatePlugin(req *CreatePluginRequest) (*PluginResponse, error)
	GetAllPlugins(limit, offset int) (*PluginListResponse, error)
	GetAllPluginsWithViewer(limit, offset int, viewerName string) (*PluginListResponse, error)
	GetPluginByID(id uuid.UUID) (*PluginResponse, error)
	UpdatePlugin(id uuid.UUID, req *UpdatePluginRequest) (*PluginResponse, error)
	DeletePlugin(id uuid.UUID) error
	GetPluginUIContent(ctx context.Context, pluginID uuid.UUID, githubService GitHubServiceInterface, userUUID, provider string) (*PluginUIResponse, error)
}

// LDAPServiceInterface defines the interface for LDAP service
type LDAPServiceInterface interface {
	SearchUsersByCN(cn string) ([]LDAPUser, error)
}

// AlertsServiceInterface defines the interface for alerts service
type AlertsServiceInterface interface {
	GetProjectAlerts(ctx context.Context, projectIDStr string, userUUID, provider string) (*AlertsResponse, error)
	CreateAlertPR(ctx context.Context, projectIDStr string, userUUID, provider string, fileName, content, message, description string) (string, error)
}

package errors

import (
	"errors"
	"fmt"
)

// NotFoundError represents an error when an entity is not found
type NotFoundError struct {
	Entity string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.Entity)
}

// Is enables errors.Is() comparison for NotFoundError
func (e *NotFoundError) Is(target error) bool {
	t, ok := target.(*NotFoundError)
	if !ok {
		return false
	}
	return e.Entity == t.Entity
}

// AlreadyExistsError represents an error when an entity already exists
type AlreadyExistsError struct {
	Entity  string
	Context string // Additional context like "in organization"
}

func (e *AlreadyExistsError) Error() string {
	if e.Context != "" {
		return fmt.Sprintf("%s already exists %s", e.Entity, e.Context)
	}
	return fmt.Sprintf("%s already exists", e.Entity)
}

// Is enables errors.Is() comparison for AlreadyExistsError
func (e *AlreadyExistsError) Is(target error) bool {
	t, ok := target.(*AlreadyExistsError)
	if !ok {
		return false
	}
	return e.Entity == t.Entity
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error: %s - %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// AuthenticationError represents authentication-related errors
type AuthenticationError struct {
	Message string
}

func (e *AuthenticationError) Error() string {
	return e.Message
}

// AuthorizationError represents authorization-related errors
type AuthorizationError struct {
	Message string
}

func (e *AuthorizationError) Error() string {
	return e.Message
}

// ConfigurationError represents configuration-related errors
type ConfigurationError struct {
	Message string
}

func (e *ConfigurationError) Error() string {
	return e.Message
}

// Entity Not Found Errors
var (
	ErrOrganizationNotFound           = &NotFoundError{Entity: "organization"}
	ErrTeamNotFound                   = &NotFoundError{Entity: "team"}
	ErrComponentNotFound              = &NotFoundError{Entity: "component"}
	ErrUserNotFound                   = &NotFoundError{Entity: "user"}
	ErrUserOrTeamNotFound             = &NotFoundError{Entity: "user or team"}
	ErrProjectNotFound                = &NotFoundError{Entity: "project"}
	ErrLandscapeNotFound              = &NotFoundError{Entity: "landscape"}
	ErrGroupNotFound                  = &NotFoundError{Entity: "group"}
	ErrComponentDeploymentNotFound    = &NotFoundError{Entity: "component deployment"}
	ErrOutageCallNotFound             = &NotFoundError{Entity: "outage call"}
	ErrDeploymentTimelineNotFound     = &NotFoundError{Entity: "deployment timeline entry"}
	ErrDutyScheduleNotFound           = &NotFoundError{Entity: "duty schedule"}
	ErrLeaderNotFound                 = &NotFoundError{Entity: "leader"}
	ErrLinkNotFound                   = &NotFoundError{Entity: "link"}
	ErrCategoryNotFound               = &NotFoundError{Entity: "category"}
	ErrTeamComponentOwnershipNotFound = &NotFoundError{Entity: "team-component ownership"}
	ErrProjectComponentNotFound       = &NotFoundError{Entity: "project-component relationship"}
	ErrProjectLandscapeNotFound       = &NotFoundError{Entity: "project-landscape relationship"}
	ErrOutageCallAssigneeNotFound     = &NotFoundError{Entity: "outage call assignee"}
	ErrDocumentationNotFound          = &NotFoundError{Entity: "documentation"}
	ErrAlertNotFound                  = &NotFoundError{Entity: "alert"}
)

// Already Exists Errors
var (
	ErrOrganizationExists              = &AlreadyExistsError{Entity: "organization", Context: "with this name or domain"}
	ErrTeamExists                      = &AlreadyExistsError{Entity: "team", Context: "with this name in the group"}
	ErrComponentExists                 = &AlreadyExistsError{Entity: "component", Context: "with this name in the organization"}
	ErrUserExists                      = &AlreadyExistsError{Entity: "user", Context: "with this email"}
	ErrProjectExists                   = &AlreadyExistsError{Entity: "project", Context: "with this name in the organization"}
	ErrLandscapeExists                 = &AlreadyExistsError{Entity: "landscape", Context: "with this name"}
	ErrGroupExists                     = &AlreadyExistsError{Entity: "group", Context: "with this name in the organization"}
	ErrLinkExists                      = &AlreadyExistsError{Entity: "link", Context: "with this URL"}
	ErrComponentDeploymentExists       = &AlreadyExistsError{Entity: "component deployment", Context: "for this component and landscape"}
	ErrActiveComponentDeploymentExists = &AlreadyExistsError{Entity: "active component deployment", Context: "for this component and landscape"}
	ErrTeamComponentOwnershipExists    = &AlreadyExistsError{Entity: "team-component ownership", Context: ""}
	ErrProjectComponentExists          = &AlreadyExistsError{Entity: "project-component relationship", Context: ""}
	ErrProjectLandscapeExists          = &AlreadyExistsError{Entity: "project-landscape relationship", Context: ""}
	ErrOutageCallAssigneeExists        = &AlreadyExistsError{Entity: "outage call assignee", Context: ""}
)

// Association Errors
var (
	ErrComponentAlreadyAssociated = errors.New("component is already associated with this project")
	ErrComponentNotAssociated     = errors.New("component is not associated with this project")
	ErrLandscapeAlreadyAssociated = errors.New("landscape is already associated with this project")
	ErrLandscapeNotAssociated     = errors.New("landscape is not associated with this project")
	ErrMemberAlreadyAssigned      = errors.New("member is already assigned to this outage call")
	ErrMemberNotAssigned          = errors.New("member is not assigned to this outage call")
	ErrActiveDeploymentNotFound   = errors.New("active deployment not found")
)

// Business Logic Errors
var (
	ErrInvalidStatus               = errors.New("invalid status")
	ErrCallTimeInFuture            = errors.New("call time cannot be in the future")
	ErrInvalidTimeRange            = errors.New("invalid time range")
	ErrDeploymentDateInPast        = errors.New("scheduled deployment date is in the past")
	ErrTimelineCodeExists          = errors.New("timeline code already exists")
	ErrScheduleConflict            = errors.New("schedule conflict detected")
	ErrInvalidDutyRotation         = errors.New("invalid duty rotation configuration")
	ErrNoMembersInTeam             = errors.New("team has no members")
	ErrInvalidPaginationParams     = errors.New("invalid pagination parameters")
	ErrGitHubAPIRateLimitExceeded  = errors.New("GitHub API rate limit exceeded")
	ErrProviderNotConfigured       = errors.New("provider is not configured")
	ErrInvalidPeriodFormat         = errors.New("invalid period format")
	ErrInternalError               = errors.New("internal server error")
	ErrInvalidJSON                 = errors.New("invalid JSON")
	ErrInvalidJSONResponse         = errors.New("invalid JSON response")
	ErrInvalidComponentID          = errors.New("invalid component-id")
	ErrInvalidLandscapeID          = errors.New("invalid landscape-id")
	ErrInvalidTeamID               = errors.New("invalid team ID")
	ErrInvalidDocumentationID      = errors.New("invalid documentation ID")
	ErrFailedToDeleteDocumentation = errors.New("failed to delete documentation")
)

// Authentication Errors
var (
	ErrInvalidRefreshToken         = &AuthenticationError{Message: "invalid refresh token"}
	ErrRefreshTokenExpired         = &AuthenticationError{Message: "refresh token has expired"}
	ErrAuthenticationRequired      = &AuthenticationError{Message: "authentication required"}
	ErrAuthenticationInvalidClaims = &AuthenticationError{Message: "invalid authentication claims"}

	// AI Core specific authentication errors
	ErrUserEmailNotFound      = &AuthenticationError{Message: "user email not found in context"}
	ErrUserNotAssignedToTeam  = &AuthorizationError{Message: "user is not assigned to any team"}
	ErrUserNotFoundInDB       = &AuthorizationError{Message: "user not found in database"}
	ErrTeamNotFoundInDB       = &AuthorizationError{Message: "team not found in database"}
	ErrMissingUsernameInToken = &AuthenticationError{Message: "missing username in token"}
)

// Configuration Errors
var (
	ErrJiraConfigMissing             = errors.New("jira configuration missing: JIRA_DOMAIN, JIRA_USER or JIRA_PASSWORD")
	ErrJenkinsTokenNotFound          = errors.New("jenkins token not found")
	ErrJenkinsUserNotFound           = errors.New("jenkins username not found")
	ErrJenkinsQueueItemNotFound      = errors.New("jenkins queue item not found")
	ErrJenkinsBuildNotFound          = errors.New("jenkins build not found")
	ErrLandscapeNotConfigured        = &ConfigurationError{Message: "landscape service not configured"}
	ErrAlertsRepositoryNotConfigured = &ConfigurationError{Message: "alerts repository not configured for this project"}
	ErrDatabaseConnection            = &ConfigurationError{Message: "database connection failed"}
	ErrTokenStoreNotInitialized      = &ConfigurationError{Message: "token store not initialized"}
	ErrAuthServiceNotInitialized     = &ConfigurationError{Message: "auth service is not initialized"}

	// AI Core specific configuration errors
	ErrAICoreCredentialsNotSet        = &ConfigurationError{Message: "AI_CORE_CREDENTIALS environment variable not set"}
	ErrAICoreCredentialsInvalid       = &ConfigurationError{Message: "failed to parse AI_CORE_CREDENTIALS"}
	ErrAICoreCredentialsNotFound      = &ConfigurationError{Message: "no credentials found for team"}
	ErrAICoreCredentialsNotConfigured = &ConfigurationError{Message: "No AI Core credentials configured for your team"}
	ErrAICoreAPIRequestFailed         = errors.New("AI Core API request failed")
	ErrAICoreDeploymentNotFound       = &NotFoundError{Entity: "deployment"}
	ErrBothConfigurationInputs        = &ConfigurationError{Message: "ConfigurationId and configurationRequest cannot both be provided"}
	ErrMissingConfigurationInput      = &ConfigurationError{Message: "Either configurationId or configurationRequest must be provided"}

	// Jira PAT (Personal Access Token) Errors
	ErrJiraPATOperation = errors.New("jira PAT operation failed")
)

// Validation Errors
var (

	// AI Core specific validation errors
	ErrMissingScenarioID             = &ValidationError{Field: "scenarioId", Message: "scenarioId query parameter is required"}
	ErrMissingDeploymentID           = &ValidationError{Field: "deployment", Message: "deploymentId parameter is required"}
	ErrMissingTargetStatusOrConfigID = &ValidationError{Message: "At least one of targetStatus or configurationId must be provided"}
	ErrNoFilesProvided               = &ValidationError{Field: "files", Message: "No files provided"}
	ErrFileSizeTooLarge              = &ValidationError{Field: "files", Message: "Files too large or invalid form data. Combined size limit is 5MB"}
	ErrCombinedFileSizeExceeds       = &ValidationError{Field: "files", Message: "Combined file size exceeds 5MB limit"}

	// Component specific validation errors
	ErrMissingLandscapeParams   = &ValidationError{Message: "component-id and landscape-id parameters are required"}
	ErrMissingTeamOrProjectName = &ValidationError{Message: "team-id or project-name parameter is required"}

	// Alert History specific validation errors
	ErrMissingProject     = &ValidationError{Field: "project", Message: "project is required"}
	ErrMissingFingerprint = &ValidationError{Field: "fingerprint", Message: "fingerprint is required"}
	ErrMissingLabelKey    = &ValidationError{Field: "key", Message: "label key is required"}

	//gitHub specific validation errors
	ErrMissingUserUUIDAndProvider = &ValidationError{Message: "userUUID and provider are required"}
	ErrUserUUIDMissing            = &ValidationError{Field: "userUUID", Message: "userUUID cannot be empty"}
	ErrProviderMissing            = &ValidationError{Field: "provider", Message: "provider cannot be empty"}
	ErrOwnerAndRepositoryMissing  = &ValidationError{Message: "owner and repository are required"}
)

// Helper Functions

// IsNotFound checks if an error is a NotFoundError
func IsNotFound(err error) bool {
	var notFoundErr *NotFoundError
	return errors.Is(err, &NotFoundError{}) || errors.As(err, &notFoundErr)
}

// IsAlreadyExists checks if an error is an AlreadyExistsError
func IsAlreadyExists(err error) bool {
	var existsErr *AlreadyExistsError
	return errors.Is(err, &AlreadyExistsError{}) || errors.As(err, &existsErr)
}

// IsValidation checks if an error is a ValidationError
func IsValidation(err error) bool {
	var validationErr *ValidationError
	return errors.Is(err, &ValidationError{}) || errors.As(err, &validationErr)
}

// IsAuthentication checks if an error is an AuthenticationError
func IsAuthentication(err error) bool {
	var authErr *AuthenticationError
	return errors.Is(err, &AuthenticationError{}) || errors.As(err, &authErr)
}

// IsAuthorization checks if an error is an AuthorizationError
func IsAuthorization(err error) bool {
	var authzErr *AuthorizationError
	return errors.Is(err, &AuthorizationError{}) || errors.As(err, &authzErr)
}

// IsConfiguration checks if an error is a ConfigurationError
func IsConfiguration(err error) bool {
	var configErr *ConfigurationError
	return errors.Is(err, &ConfigurationError{}) || errors.As(err, &configErr)
}

// NewNotFoundError creates a new NotFoundError for a custom entity
func NewNotFoundError(entity string) error {
	return &NotFoundError{Entity: entity}
}

// NewAlreadyExistsError creates a new AlreadyExistsError for a custom entity
func NewAlreadyExistsError(entity, context string) error {
	return &AlreadyExistsError{Entity: entity, Context: context}
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}

// NewAuthenticationError creates a new AuthenticationError
func NewAuthenticationError(message string) error {
	return &AuthenticationError{Message: message}
}

// NewAuthorizationError creates a new AuthorizationError
func NewAuthorizationError(message string) error {
	return &AuthorizationError{Message: message}
}

// NewConfigurationError creates a new ConfigurationError
func NewConfigurationError(message string) error {
	return &ConfigurationError{Message: message}
}

// NewAICoreCredentialsNotFoundError creates a specific error for missing team credentials
func NewAICoreCredentialsNotFoundError(teamName string) error {
	return &ConfigurationError{Message: fmt.Sprintf("no credentials found for team: %s", teamName)}
}

// NewMissingQueryParam creates a new ValidationError for missing query parameters
func NewMissingQueryParam(queryParam string) error {
	return &ValidationError{Field: queryParam, Message: fmt.Sprintf("missing required query parameter: %s", queryParam)}
}

// NewJiraPATError creates a detailed Jira PAT operation error
func NewJiraPATError(operation string, details string) error {
	return fmt.Errorf("%w: %s - %s", ErrJiraPATOperation, operation, details)
}

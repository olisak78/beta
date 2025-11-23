package routes

import (
	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/api/middleware"
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/config"
	"developer-portal-backend/internal/repository"
	"developer-portal-backend/internal/service"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

// userRepoAdapter adapts repository.MemberRepository to auth.MemberRepository
type userRepoAdapter struct {
	repo *repository.UserRepository
}

func (a *userRepoAdapter) GetByEmail(email string) (interface{}, error) {
	return a.repo.GetByEmail(email)
}

// SetupRoutes configures all the routes for the application
func SetupRoutes(db *gorm.DB, cfg *config.Config) *gin.Engine {
	// Create router
	router := gin.New()

	// Add middleware
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.CORS(cfg))

	// Initialize validator
	validator := validator.New()

	// Initialize repositories
	organizationRepo := repository.NewOrganizationRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	projectRepo := repository.NewProjectRepository(db)
	componentRepo := repository.NewComponentRepository(db)
	landscapeRepo := repository.NewLandscapeRepository(db)
	categoryRepo := repository.NewCategoryRepository(db)
	linkRepo := repository.NewLinkRepository(db)
	docRepo := repository.NewDocumentationRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo, linkRepo, validator)
	teamService := service.NewTeamService(teamRepo, groupRepo, organizationRepo, userRepo, linkRepo, componentRepo, validator)
	projectService := service.NewProjectService(projectRepo, validator)
	componentService := service.NewComponentService(componentRepo, organizationRepo, projectRepo, validator)
	landscapeService := service.NewLandscapeService(landscapeRepo, organizationRepo, projectRepo, validator)
	categoryService := service.NewCategoryService(categoryRepo, validator)
	linkService := service.NewLinkService(linkRepo, userRepo, teamRepo, categoryRepo, validator)
	docService := service.NewDocumentationService(docRepo, teamRepo, validator)
	ldapService := service.NewLDAPService(cfg)
	jiraService := service.NewJiraService(cfg)
	// Initialize Jira PAT on startup: use fixed-name PAT with machine identifier, delete existing if present, then create a new one
	if err := jiraService.InitializePATOnStartup(); err != nil {
		log.Printf("Warning: Jira PAT initialization failed: %v", err)
	}
	jenkinsService := service.NewJenkinsService(cfg)
	sonarService := service.NewSonarService(cfg)
	aicoreService := service.NewAICoreService(userRepo, teamRepo, groupRepo, organizationRepo)

	// Initialize auth configuration and services
	authConfig, err := auth.LoadAuthConfig("config/auth.yaml")
	if err != nil {
		log.Printf("Warning: Failed to load auth config: %v", err)
		// Continue without auth if config fails to load
		authConfig = nil
	}

	var authHandler *auth.AuthHandler
	var authMiddleware *auth.AuthMiddleware
	var authService *auth.AuthService
	if authConfig != nil {
		memberRepoAuth := &userRepoAdapter{repo: userRepo}
		authService, err = auth.NewAuthService(authConfig, memberRepoAuth)
		if err != nil {
			log.Printf("Warning: Failed to initialize auth service: %v", err)
		} else {
			authHandler = auth.NewAuthHandler(authService)
			authMiddleware = auth.NewAuthMiddleware(authService)
		}
	}

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	userHandler := handlers.NewUserHandler(userService, teamRepo)
	teamHandler := handlers.NewTeamHandler(teamService)
	projectHandler := handlers.NewProjectHandler(projectService)
	componentHandler := handlers.NewComponentHandler(componentService, teamService)
	landscapeHandler := handlers.NewLandscapeHandler(landscapeService)
	categoryHandler := handlers.NewCategoryHandler(categoryService)
	linkHandler := handlers.NewLinkHandler(linkService)
	docHandler := handlers.NewDocumentationHandler(docService)
	ldapHandler := handlers.NewLDAPHandler(ldapService, userRepo)
	jiraHandler := handlers.NewJiraHandler(jiraService)
	jenkinsHandler := handlers.NewJenkinsHandler(jenkinsService)
	sonarHandler := handlers.NewSonarHandler(sonarService)
	githubService := service.NewGitHubService(authService)
	githubHandler := handlers.NewGitHubHandler(githubService)
	aicoreHandler := handlers.NewAICoreHandler(aicoreService, validator)
	alertsService := service.NewAlertsService(projectRepo, authService)
	alertsHandler := handlers.NewAlertsHandler(alertsService)

	// Health check routes
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/health/live", healthHandler.Live)

	// Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Auth routes (Backstage-compatible)
	if authHandler != nil {
		auth := router.Group("/api/auth")
		{
			// Provider-specific auth routes
			providerGroup := auth.Group("/:provider")
			{
				providerGroup.GET("/start", authHandler.Start)
				providerGroup.GET("/handler/frame", authHandler.HandlerFrame)
				providerGroup.GET("/refresh", authHandler.Refresh)
				providerGroup.POST("/logout", authHandler.Logout)
			}

			// Helper endpoint for token validation (not part of Backstage spec)
			auth.POST("/validate", authHandler.ValidateToken)
		}
	}

	// API v1 routes - All endpoints require authentication
	v1 := router.Group("/api/v1")

	// Auth middleware is mandatory - endpoints rely on user context
	if authMiddleware == nil {
		panic("Authentication middleware is required but not initialized. Check auth configuration.")
	}
	v1.Use(authMiddleware.RequireAuth())

	{

		// Users routes
		users := v1.Group("/users")
		{
			users.GET("/search/new", ldapHandler.UserSearch)
			users.POST("", userHandler.CreateUser)
			users.PUT("", userHandler.UpdateUserTeam)
			users.GET("", userHandler.ListUsers)
			users.GET("/:user_id", userHandler.GetMemberByUserID)
			users.POST("/:user_id/favorites/:link_id", userHandler.AddFavoriteLink)
			users.DELETE("/:user_id/favorites/:link_id", userHandler.RemoveFavoriteLink)
		}

		// Current user route: /users/me
		v1.GET("/users/me", userHandler.GetCurrentUser)

		// Team routes
		teams := v1.Group("/teams")
		{
			teams.GET("", teamHandler.GetAllTeams)
			teams.PATCH("/:id/metadata", teamHandler.UpdateTeamMetadata)           // Update team metadata
			teams.GET("/:id/documentations", docHandler.GetDocumentationsByTeamID) // Get documentations by team ID
		}

		// Documentation routes
		documentations := v1.Group("/documentations")
		{
			documentations.POST("", docHandler.CreateDocumentation)
			documentations.GET("/:id", docHandler.GetDocumentationByID)
			documentations.PATCH("/:id", docHandler.UpdateDocumentation)
			documentations.DELETE("/:id", docHandler.DeleteDocumentation)
		}

		// Project routes
		projects := v1.Group("/projects")
		{
			projects.GET("", projectHandler.GetAllProjects)
		}

		// Component routes
		components := v1.Group("/components")
		{
			components.GET("", componentHandler.ListComponents)
		}

		// Query-param endpoint: /api/v1/landscapes?project-name=<project_name>
		v1.GET("/landscapes", landscapeHandler.ListLandscapesByQuery)

		// CIS public endpoints proxy: /api/v1/cis-public/proxy?url=<component_public_url>
		// Used for proxying health checks, version info, and other public endpoints
		v1.GET("/cis-public/proxy", healthHandler.ProxyComponentHealth)

		// Jira routes - Consolidated endpoints
		jira := v1.Group("/jira")
		{
			jira.GET("/issues", jiraHandler.GetIssues)                 // GET /jira/issues?project=SAPBTPCFS&status=Open,In Progress&team=MyTeam
			jira.GET("/issues/me", jiraHandler.GetMyIssues)            // GET /jira/issues/me?status=Open&count_only=true
			jira.GET("/issues/me/count", jiraHandler.GetMyIssuesCount) // GET /jira/issues/me/count?status=Resolved&date=2023-01-01
		}

		// GitHub routes
		github := v1.Group("/github")
		{
			github.GET("/pull-requests", githubHandler.GetMyPullRequests)
			github.PATCH("/pull-requests/close/:pr_number", githubHandler.ClosePullRequest)
			github.GET("/prs", githubHandler.GetMyPullRequests) // Convenient alias
			github.GET("/contributions", githubHandler.GetUserTotalContributions)
			github.GET("/average-pr-time", githubHandler.GetAveragePRMergeTime)
			github.GET("/pr-review-comments", githubHandler.GetPRReviewComments)
			github.GET("/:provider/heatmap", githubHandler.GetContributionsHeatmap)
			// Repository content proxy for documentation viewer
			github.GET("/repos/:owner/:repo/contents/*path", githubHandler.GetRepositoryContent)
			github.PUT("/repos/:owner/:repo/contents/*path", githubHandler.UpdateRepositoryFile)
			// GitHub asset proxy for images and other assets
			github.GET("/asset", githubHandler.GetGitHubAsset)
		}

		// Sonar routes
		sonar := v1.Group("/sonar")
		{
			sonar.GET("/measures", sonarHandler.GetMeasures)
		}

		// Self-service routes (for Jenkins, and future services like Kubernetes, etc.)
		selfService := v1.Group("/self-service")
		{
			// Jenkins self-service endpoints
			jenkins := selfService.Group("/jenkins")
			{
				jenkins.GET("/:jaasName/:jobName/parameters", jenkinsHandler.GetJobParameters)
				jenkins.POST("/:jaasName/:jobName/trigger", jenkinsHandler.TriggerJob)
				jenkins.GET("/:jaasName/queue/:queueItemId/status", jenkinsHandler.GetQueueItemStatus)
				jenkins.GET("/:jaasName/:jobName/:buildNumber/status", jenkinsHandler.GetBuildStatus)
			}
		}

		// AI Core routes
		aicore := v1.Group("/ai-core")
		{
			aicore.GET("/deployments", aicoreHandler.GetDeployments)
			aicore.GET("/deployments/:deploymentId", aicoreHandler.GetDeploymentDetails)
			aicore.GET("/models", aicoreHandler.GetModels)
			aicore.GET("/me", aicoreHandler.GetMe)
			aicore.POST("/configurations", aicoreHandler.CreateConfiguration)
			aicore.POST("/deployments", aicoreHandler.CreateDeployment)
			aicore.PATCH("/deployments/:deploymentId", aicoreHandler.UpdateDeployment)
			aicore.DELETE("/deployments/:deploymentId", aicoreHandler.DeleteDeployment)
			aicore.POST("/chat/inference", aicoreHandler.ChatInference)
			aicore.POST("/upload", aicoreHandler.UploadAttachment)
		}

		// Alerts routes - Prometheus AlertManager alerts from GitHub
		alerts := v1.Group("/projects/:projectId/alerts")
		{
			alerts.GET("", alertsHandler.GetAlerts)         // GET /api/v1/projects/:projectId/alerts
			alerts.POST("/pr", alertsHandler.CreateAlertPR) // POST /api/v1/projects/:projectId/alerts/pr
		}

		// Category routes
		categories := v1.Group("/categories")
		{
			categories.GET("", categoryHandler.ListCategories)
		}

		// Link routes
		links := v1.Group("/links")
		{
			links.GET("", linkHandler.ListLinks) // GET /api/v1/links?owner=<user_id>
			links.POST("", linkHandler.CreateLink)
			links.DELETE("/:id", linkHandler.DeleteLink)
		}

		// Nested resource routes moved to respective groups to avoid conflicts
		// Landscape-specific component deployments route moved to landscapes group
	}

	// Catch-all route for undefined endpoints
	router.NoRoute(func(c *gin.Context) {
		c.JSON(404, gin.H{
			"error":      "Endpoint not found",
			"path":       c.Request.URL.Path,
			"method":     c.Request.Method,
			"request_id": c.GetString("request_id"),
		})
	})

	return router
}

// SetupHealthRoutes sets up only health check routes (useful for testing)
func SetupHealthRoutes(db *gorm.DB) *gin.Engine {
	router := gin.New()
	router.Use(middleware.Logger())
	router.Use(middleware.Recovery())

	healthHandler := handlers.NewHealthHandler(db)
	router.GET("/health", healthHandler.Health)
	router.GET("/health/ready", healthHandler.Ready)
	router.GET("/health/live", healthHandler.Live)

	return router
}

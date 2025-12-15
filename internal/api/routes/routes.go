package routes

import (
	"developer-portal-backend/internal/api/handlers"
	"developer-portal-backend/internal/api/middleware"
	"developer-portal-backend/internal/auth"
	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/client"
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

	// Initialize cache service
	cacheConfig := cache.DefaultCacheConfig()
	cacheService := cache.NewInMemoryCache(cacheConfig)
	ttlConfig := cache.DefaultTTLConfig()

	log.Printf("Cache service initialized: enabled=%v, defaultTTL=%v", cacheConfig.Enabled, cacheConfig.DefaultTTL)

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
	tokenRepo := repository.NewTokenRepository(db)
	pluginRepo := repository.NewPluginRepository(db)

	// Initialize services
	userService := service.NewUserService(userRepo, linkRepo, pluginRepo, validator)
	teamService := service.NewTeamService(teamRepo, groupRepo, organizationRepo, userRepo, linkRepo, componentRepo, validator)
	projectService := service.NewProjectService(projectRepo, validator)
	componentService := service.NewComponentService(componentRepo, organizationRepo, projectRepo, validator)

	// Initialize landscape service with caching
	landscapeService := service.NewLandscapeServiceWithCache(
		landscapeRepo,
		organizationRepo,
		projectRepo,
		validator,
		cacheService,
		ttlConfig,
	)

	categoryService := service.NewCategoryService(categoryRepo, validator)
	linkService := service.NewLinkService(linkRepo, userRepo, teamRepo, categoryRepo, validator)
	docService := service.NewDocumentationService(docRepo, teamRepo, validator)

	pluginService := service.NewPluginService(pluginRepo, userRepo, validator)

	// Initialize auth configuration and services after service initialization
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
		userRepoAuth := &userRepoAdapter{repo: userRepo}
		authService, err = auth.NewAuthService(authConfig, userRepoAuth, tokenRepo)
		if err != nil {
			log.Printf("Warning: Failed to initialize auth service: %v", err)
		} else {
			authHandler = auth.NewAuthHandler(authService)
			authMiddleware = auth.NewAuthMiddleware(authService)
		}
	}

	ldapService := service.NewLDAPService(cfg)
	jiraService := service.NewJiraService(cfg)
	// Initialize Jira PAT on startup: use fixed-name PAT with machine identifier, delete existing if present, then create a new one
	if err := jiraService.InitializePATOnStartup(); err != nil {
		log.Printf("Warning: Jira PAT initialization failed: %v", err)
	}
	jenkinsService := service.NewJenkinsService(cfg)
	sonarService := service.NewSonarService(cfg)
	aicoreService := service.NewAICoreService(userRepo, teamRepo, groupRepo, organizationRepo)

	// Initialize alert history client and service
	alertHistoryClient := client.NewAlertHistoryClient(cfg.MonitoringServiceURL)
	alertHistoryService := service.NewAlertHistoryService(alertHistoryClient)

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(db)
	userHandler := handlers.NewUserHandler(userService, teamService)
	teamHandler := handlers.NewTeamHandler(teamService)
	projectHandler := handlers.NewProjectHandler(projectService)
	componentHandler := handlers.NewComponentHandler(componentService, landscapeService, teamService, projectService)
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
	pluginHandler := handlers.NewPluginHandlerWithGitHub(pluginService, githubService)
	aicoreHandler := handlers.NewAICoreHandler(aicoreService, validator)
	alertsService := service.NewAlertsService(projectRepo, authService)
	alertsHandler := handlers.NewAlertsHandler(alertsService)
	alertHistoryHandler := handlers.NewAlertHistoryHandler(alertHistoryService)

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
			auth.GET("/refresh", authHandler.Refresh)
			auth.POST("/logout", authHandler.Logout)
			providerGroup := auth.Group("/:provider")
			{
				providerGroup.GET("/start", authHandler.Start)
				providerGroup.GET("/handler/frame", authHandler.HandlerFrame)
			}

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
			users.POST("/:user_id/plugins/:plugin_id", userHandler.AddSubscribedPlugin)
			users.DELETE("/:user_id/plugins/:plugin_id", userHandler.RemoveSubscribedPlugin)
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
			components.GET("/health", componentHandler.ComponentHealth) // GET /api/v1/components/health?component-id=<>&landscape-id=<>
		}

		// Landscapes routes
		landscapes := v1.Group("/landscapes")
		{
			landscapes.GET("", landscapeHandler.ListLandscapesByQuery)  // GET /api/v1/landscapes?project-name=<project_name>
			landscapes.DELETE("/:id", landscapeHandler.DeleteLandscape) // DELETE /api/v1/landscapes/:id
		}

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
			// Jenkins routes
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
			// Deployment management
			aicore.GET("/deployments", aicoreHandler.GetDeployments)
			aicore.GET("/deployments/:deploymentId", aicoreHandler.GetDeploymentDetails)
			aicore.POST("/deployments", aicoreHandler.CreateDeployment)
			aicore.PATCH("/deployments/:deploymentId", aicoreHandler.UpdateDeployment)
			aicore.DELETE("/deployments/:deploymentId", aicoreHandler.DeleteDeployment)

			// Model and configuration management
			aicore.GET("/models", aicoreHandler.GetModels)
			aicore.GET("/me", aicoreHandler.GetMe)
			aicore.POST("/configurations", aicoreHandler.CreateConfiguration)

			// Chat inference
			aicore.POST("/chat/inference", aicoreHandler.ChatInference)

			// Attachment management
			aicore.POST("/upload", aicoreHandler.UploadAttachment)
		}

		// Alerts routes
		alerts := v1.Group("/projects/:projectId/alerts")
		{
			alerts.GET("", alertsHandler.GetAlerts)
			alerts.POST("/pr", alertsHandler.CreateAlertPR)
		}

		// Alert history routes
		alertHistory := v1.Group("/alert-history")
		{
			alertHistory.GET("/projects", alertHistoryHandler.GetAvailableProjects)                       // GET /api/v1/alert-history/projects
			alertHistory.GET("/alerts/:project", alertHistoryHandler.GetAlertsByProject)                  // GET /api/v1/alert-history/alerts/:project?page=1&pageSize=50&severity=critical&status=firing
			alertHistory.GET("/alerts/:project/filters", alertHistoryHandler.GetAlertFilters)             // GET /api/v1/alert-history/alerts/:project/filters?severity=critical&landscape=production
			alertHistory.GET("/alerts/:project/:fingerprint", alertHistoryHandler.GetAlertByFingerprint)  // GET /api/v1/alert-history/alerts/:project/:fingerprint
			alertHistory.PUT("/alerts/:project/:fingerprint/label", alertHistoryHandler.UpdateAlertLabel) // PUT /api/v1/alert-history/alerts/:project/:fingerprint/label
		}

		// Link routes
		links := v1.Group("/links")
		{
			links.GET("", linkHandler.ListLinks)
			links.POST("", linkHandler.CreateLink)
			links.PUT("/:id", linkHandler.UpdateLink)
			links.DELETE("/:id", linkHandler.DeleteLink)

		}

		// Category routes
		categories := v1.Group("/categories")
		{
			categories.GET("", categoryHandler.ListCategories)
		}

		// Plugin routes
		plugins := v1.Group("/plugins")
		{
			plugins.POST("", pluginHandler.CreatePlugin)                // POST /api/v1/plugins
			plugins.GET("", pluginHandler.GetAllPlugins)                // GET /api/v1/plugins
			plugins.GET("/:id", pluginHandler.GetPluginByID)            // GET /api/v1/plugins/{id}
			plugins.PUT("/:id", pluginHandler.UpdatePlugin)             // PUT /api/v1/plugins/{id}
			plugins.DELETE("/:id", pluginHandler.DeletePlugin)          // DELETE /api/v1/plugins/{id}
			plugins.GET("/:id/ui", pluginHandler.GetPluginUI)           // GET /api/v1/plugins/{id}/ui
			plugins.GET("/:id/proxy", pluginHandler.ProxyPluginBackend) // GET /api/v1/plugins/{id}/proxy?path={targetPath}
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

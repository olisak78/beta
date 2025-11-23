# Developer Portal Backend

A robust Go backend server for a developer portal, built with modern best practices and clean architecture.

## 🚀 TLDR - Quick Start

```bash
# 1. Clone and setup
git clone <repository-url>
cd developer-portal-backend
make setup

# 2. Setup environment configuration
cp .env.example .env
# Edit .env to configure port and other settings

# 3. Start development server (auto-generates JWT secret)
make run-dev

# 4. Server runs on http://localhost:7008 (default) or PORT from .env
```

**That's it!** The server will be running with a test database and sample data loaded.

---

## 🏗️ Architecture

This project follows **Clean Architecture** principles with clear separation of concerns:

```
developer-portal-backend/
├── cmd/
│   └── server/                 # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/          # HTTP handlers (controllers)
│   │   ├── middleware/        # Custom middleware
│   │   └── routes/            # Route definitions
│   ├── auth/                  # Authentication (GitHub OAuth)
│   ├── config/                # Configuration management
│   ├── database/
│   │   ├── migrations/        # SQL migration files
│   │   └── models/            # Database models/entities
│   ├── mocks/                 # Test mocks
│   ├── repository/            # Data access layer
│   ├── service/               # Business logic layer
│   ├── testutils/            # Testing utilities
│   └── utils/                 # Utility functions
├── config/                    # Configuration files
├── docker/                    # Docker configuration
├── pkg/                       # Public packages
├── scripts/                   # Development scripts
├── .env.example              # Environment variables template
├── Makefile                  # Development tasks
└── README.md
```

## 🚀 Setup & Running

### Option 1: Quick Development Setup (Recommended)

```bash
# Complete setup with one command
make setup

# Start development server (auto-generates JWT secret and loads sample data)
make run-dev
```

### Option 2: Manual Setup

```bash
# 1. Install dependencies
make deps

# 2. Start database
make db-up

# 3. Setup environment
cp .env.example .env
# Edit .env as needed

# 4. Run migrations (if any)
make migrate-up

# 5. Start server
make run-dev

# Initial data
# Loaded automatically on server startup from scripts/data/*.yaml
```

### Option 3: Production Mode

```bash
# Set JWT secret
export JWT_SECRET=$(openssl rand -base64 32)

# Build and run
make run
```

The server will start on `http://localhost:7008` (configurable via PORT in .env)

## 🐳 Docker Development

### Start All Services

```bash
# Start PostgreSQL, pgAdmin, and Redis
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

### Database Management

```bash
# Start only PostgreSQL
make db-up

# Reset database (WARNING: deletes all data)
make db-reset

# Initial data
# Loaded automatically on server startup from scripts/data/*.yaml
```

## 🗄️ Database Access

- **PostgreSQL**: `localhost:5432`
  - Database: `developer_portal`
  - Username: `postgres`
  - Password: `postgres`

- **pgAdmin**: `http://localhost:5050`
  - Email: `admin@developer-portal.com`
  - Password: `admin`

## 🔐 Authentication

The application supports **GitHub OAuth authentication** with multi-provider configuration:

### Supported Providers
- **githubtools**: GitHub Enterprise (https://github.tools.sap)
- **githubwdf**: GitHub Enterprise (https://github.wdf.sap.corp)

### Auth Configuration
Edit `config/auth.yaml` and set environment variables:
```bash
# For GitHub Tools
GITHUB_TOOLS_APP_CLIENT_ID=your_client_id
GITHUB_TOOLS_APP_CLIENT_SECRET=your_client_secret

# For GitHub WDF
GITHUB_WDF_APP_CLIENT_ID=your_client_id
GITHUB_WDF_APP_CLIENT_SECRET=your_client_secret
```

### Auth Flow (Backstage Compatible)
1. `GET /api/auth/{provider}/start` - Initiate OAuth flow
2. `GET /api/auth/{provider}/handler/frame` - Handle OAuth callback
3. `GET /api/auth/{provider}/refresh` - Refresh tokens
4. `POST /api/auth/{provider}/logout` - Logout

## 📡 API Endpoints

### Health Checks
- `GET /health` - Application health status
- `GET /health/ready` - Readiness check
- `GET /health/live` - Liveness check


### Teams API (v1)
- `GET /api/v1/teams` - Get teams or a specific team using query parameters:
  - `team-name` (string): returns a single team enriched with members and links
  - `team-id` (UUID): returns a single team enriched with members and links
  - no query: returns a list of teams (id, group_id, name, title, description, picture_url)

### Users API (v1)
- `GET /api/v1/users` - List users
- `GET /api/v1/users/me` - Get current user
- `GET /api/v1/users/search/new` - Search LDAP users
- `POST /api/v1/users` - Create user
- `PUT /api/v1/users` - Update user team
- `GET /api/v1/users/:user_id` - Get user by user ID
- `POST /api/v1/users/:user_id/favorites/:link_id` - Add a favorite link
- `DELETE /api/v1/users/:user_id/favorites/:link_id` - Remove a favorite link


### Components API (v1)
- `GET /api/v1/components` - List components filtered by either:
  - `team-id` (UUID): components owned by the given team
  - `project-name` (string): all components for the given project
  One of `team-id` or `project-name` is required.

### Landscapes API (v1)
- `GET /api/v1/landscapes` - List landscapes by query parameters

### Documentations API (v1)
- `POST /api/v1/documentations` - Create documentation
- `GET /api/v1/documentations/:id` - Get documentation by ID
- `PATCH /api/v1/documentations/:id` - Update documentation
- `DELETE /api/v1/documentations/:id` - Delete documentation

### Jira API (v1)
- `GET /api/v1/jira/issues` - List issues (supports filters via query)
- `GET /api/v1/jira/issues/me` - List my issues
- `GET /api/v1/jira/issues/me/count` - Get my issues count

### GitHub API (v1)
- `GET /api/v1/github/pull-requests` - Get my open pull requests
- `GET /api/v1/github/prs` - Alias for pull requests
- `GET /api/v1/github/contributions` - Get user total contributions
- `GET /api/v1/github/average-pr-time` - Get average PR merge time
- `GET /api/v1/github/:provider/heatmap` - Get contributions heatmap
- `GET /api/v1/github/repos/:owner/:repo/contents/*path` - Get repository content
- `PUT /api/v1/github/repos/:owner/:repo/contents/*path` - Update repository file
- `GET /api/v1/github/asset` - Proxy GitHub asset

### AI Core API (v1)
- `GET /api/v1/ai-core/deployments` - List AI Core deployments
- `GET /api/v1/ai-core/deployments/:deploymentId` - Get deployment details
- `GET /api/v1/ai-core/models` - List available models
- `GET /api/v1/ai-core/me` - Get current AI Core user
- `POST /api/v1/ai-core/configurations` - Create configuration
- `POST /api/v1/ai-core/deployments` - Create deployment
- `PATCH /api/v1/ai-core/deployments/:deploymentId` - Update deployment
- `DELETE /api/v1/ai-core/deployments/:deploymentId` - Delete deployment
- `POST /api/v1/ai-core/chat/inference` - Chat inference
- `POST /api/v1/ai-core/upload` - Upload attachment

### Sonar API (v1)
- `GET /api/v1/sonar/measures` - Get Sonar measures for a component

### Self-service Jenkins API (v1)
- `GET /api/v1/self-service/jenkins/:jaasName/:jobName/parameters` - Get job parameters
- `POST /api/v1/self-service/jenkins/:jaasName/:jobName/trigger` - Trigger job
- `GET /api/v1/self-service/jenkins/:jaasName/queue/:queueItemId/status` - Get queue item status
- `GET /api/v1/self-service/jenkins/:jaasName/:jobName/:buildNumber/status` - Get build status

### Categories API (v1)
- `GET /api/v1/categories` - List categories

### Links API (v1)
- `GET /api/v1/links` - List links (filter by owner)
- `POST /api/v1/links` - Create link
- `DELETE /api/v1/links/:id` - Delete link

### Example API Usage

```bash
# Health check
curl http://localhost:7008/health

# Get a specific team by name
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:7008/api/v1/teams?team-name=platform-team"

# Get a specific team by ID
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:7008/api/v1/teams?team-id=00000000-0000-0000-0000-000000000000"

# List components by team-id
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:7008/api/v1/components?team-id=00000000-0000-0000-0000-000000000000"

# List components by project-name
curl -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  "http://localhost:7008/api/v1/components?project-name=my-project"
```

## 🧪 Testing

### Quick Testing (Recommended)
```bash
# Run all tests (excludes scripts directory)
make test

# Run tests with coverage
make test-coverage
```

> **Note:** Always use `make test` instead of `go test ./...` directly, as the Makefile properly excludes the scripts directory.

### Additional Testing
```bash
# Run unit tests only (fast)
make test-unit

# Pre-commit checks (format, lint, unit tests)
make test-precommit
```

### Comprehensive Testing
```bash
# Setup test environment
make test-setup

# Run all tests (unit + integration)
make test-all

# Run tests with coverage
make test-coverage-full

# Run tests with race detection
make test-race

# Cleanup test environment
make test-teardown
```

### Docker Testing
```bash
# Run tests in Docker container
make test-docker

# Start Docker test environment
make test-docker-up
```

### Generate Test Mocks
```bash
# Generate mocks for testing
make mocks
```

## 🔧 Development Commands

```bash
# Show all available commands
make help

# Development workflow
make format          # Format code
make lint           # Run linter
make build          # Build application
make clean          # Clean build artifacts

# Hot reload development (requires air)
make dev

# Database migrations
make migrate-create NAME=migration_name
make migrate-up
make migrate-down
```

## 📚 API Documentation

### Swagger/OpenAPI Documentation

The API is fully documented using Swagger/OpenAPI specifications. 

**Access Swagger UI:**
- Start the server: `make run-dev` or `go run cmd/server/main.go`
- Open: `http://localhost:7008/swagger/index.html`

**Regenerate Documentation:**
After adding or modifying API endpoint annotations in handlers, regenerate the docs:

```bash
# Regenerate swagger documentation
/Users/i572719/go/bin/swag init -g cmd/server/main.go
```

*Note: This updates `docs/docs.go`, `docs/swagger.json`, and `docs/swagger.yaml` files.*

### Authentication Endpoints

The following authentication endpoints are documented in Swagger:

- `GET /api/auth/{provider}/start` - Start OAuth authentication flow
- `GET /api/auth/{provider}/handler/frame` - Handle OAuth callback  
- `GET /api/auth/{provider}/refresh` - Refresh authentication token
- `POST /api/auth/{provider}/logout` - Logout user
- `POST /api/auth/validate` - Validate JWT token

All authenticated API endpoints (under `/api/v1/`) require a valid JWT token.

## 📝 Configuration

Configuration is managed through environment variables and YAML files.

### Environment Variables (.env)

Copy the example file and customize as needed:
```bash
cp .env.example .env
```

Available environment variables:
```bash
# Server Configuration (defaults shown)
ENVIRONMENT=development
PORT=7008
LOG_LEVEL=info

# Database Configuration
DATABASE_URL=postgres://postgres:postgres@localhost:5432/developer_portal?sslmode=disable
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=developer_portal

# JWT Configuration
JWT_SECRET=your-secret-key-change-in-production

# CORS Configuration
ALLOWED_ORIGINS=http://localhost:3000,http://localhost:8080

# GitHub OAuth (set these for authentication)
GITHUB_TOOLS_APP_CLIENT_ID=
GITHUB_TOOLS_APP_CLIENT_SECRET=
GITHUB_WDF_APP_CLIENT_ID=
GITHUB_WDF_APP_CLIENT_SECRET=
```

### Auth Configuration (config/auth.yaml)
```yaml
default_environment: "production"
redirect_url: "http://localhost:7008"  # Should match your PORT setting

providers:
  githubtools:
    environments:
      development:
        client_id: "${GITHUB_TOOLS_APP_CLIENT_ID_LOCAL}"
        client_secret: "${GITHUB_TOOLS_APP_CLIENT_SECRET_LOCAL}"
        enterprise_base_url: "https://github.tools.sap"
      production:
        client_id: "${GITHUB_TOOLS_APP_CLIENT_ID}"
        client_secret: "${GITHUB_TOOLS_APP_CLIENT_SECRET}"
        enterprise_base_url: "https://github.tools.sap"
```

## 🏗️ Project Structure Explained

This project follows **Clean Architecture** principles with clear separation of concerns:

### Core Layers

#### 📁 `cmd/server/` - Application Entry Point
- Contains `main.go` - bootstraps the application
- Loads configuration, establishes connections, starts server
- **Rule**: Only bootstrap code, no business logic

#### 📁 `internal/api/handlers/` - HTTP Controllers
- Handle HTTP requests/responses, input validation
- Delegate business logic to services
- **Rule**: Thin layer - only HTTP concerns

#### 📁 `internal/service/` - Business Logic Layer
- Core business rules, validation, orchestration
- Independent of HTTP and database details
- **Rule**: Contains the application's business logic

#### 📁 `internal/repository/` - Data Access Layer
- Database operations, CRUD, queries
- Maps between database and Go structs
- **Rule**: Only data access, no business rules

#### 📁 `internal/database/models/` - Database Entities
- Database schema, relationships, validation
- GORM models with proper associations
- **Rule**: Represents database structure

### Supporting Components

#### 📁 `internal/auth/` - Authentication
- GitHub OAuth integration
- JWT token management
- Backstage-compatible auth flow

#### 📁 `internal/config/` - Configuration
- Environment variable loading
- Configuration validation
- Centralized config access

#### 📁 `internal/testutils/` - Testing Utilities
- Test helpers, factories, HTTP utilities
- Database test setup and teardown

#### 📁 `internal/mocks/` - Test Mocks
- Generated mocks for testing
- Service and repository mocks

## 🔄 Data Flow Architecture

```
HTTP Request → Auth Middleware → Handler → Service → Repository → Database
                     ↓             ↓         ↓          ↓
HTTP Response ← Auth Middleware ← Handler ← Service ← Repository ← Database
```

### Request Flow:
1. **HTTP Request** arrives
2. **Auth Middleware** validates JWT token
3. **Handler** parses request, validates input
4. **Service** applies business logic
5. **Repository** performs database operations
6. **Database** stores/retrieves data
7. Response flows back through same layers

## 🔒 Security Considerations

- **JWT Authentication**: All API endpoints require valid JWT tokens
- **GitHub OAuth**: Secure authentication flow
- **Environment Variables**: Secrets managed through environment variables
- **CORS**: Configurable allowed origins
- **Input Validation**: Comprehensive request validation
- **SQL Injection Protection**: GORM provides protection

### Production Security Checklist
- [ ] Set secure `JWT_SECRET` (32+ random bytes)
- [ ] Configure production GitHub OAuth apps
- [ ] Restrict `ALLOWED_ORIGINS` to production domains
- [ ] Use HTTPS in production
- [ ] Secure database credentials
- [ ] Enable request logging and monitoring

## 🚀 Production Deployment

### Deployment Flow

Follow these steps in order for a complete deployment:

#### 1. Update the Chart Version
Edit `charts/developer-portal-backend/Chart.yaml` and update the `appVersion` tag:
```yaml
appVersion: "1.0.8"  # Update to your new version
```

#### 2. Build Docker Images
```bash
make docker-build
```
This builds the backend Docker image with the version from Chart.yaml. Initial data is loaded automatically by the server at startup; there is no separate init-data image or job.

#### 3. Push Images to Registry
```bash
make docker-push
```
Pushes the backend image to the Docker registry.

#### 4. Deploy to Dev Environment
**Important:** Make sure you're connected to the dev cluster first!
```bash
# Verify you're on the correct cluster
kubectl config current-context

# Deploy to dev
make deploy-dev
```

#### 5. Deploy to Prod Environment
**Important:** Make sure you're connected to the prod cluster first!
```bash
# Verify you're on the correct cluster
kubectl config current-context

# Deploy to prod
make deploy-prod
```

### Quick Commands Summary
```bash
# Complete deployment workflow
# 1. Update tag in charts/developer-portal-backend/Chart.yaml
# 2. Build images
make docker-build

# 3. Push images
make docker-push

# 4. Connect to dev cluster, then deploy
make deploy-dev

# 5. Connect to prod cluster, then deploy
make deploy-prod
```

### Environment-Specific GitHub OAuth

The deployment supports separate GitHub OAuth apps for dev and prod environments:

**Local Development** (no suffix):
```bash
export GITHUB_TOOLS_APP_CLIENT_ID=your-local-client-id
export GITHUB_TOOLS_APP_CLIENT_SECRET=your-local-client-secret
export GITHUB_WDF_APP_CLIENT_ID=your-local-wdf-id
export GITHUB_WDF_APP_CLIENT_SECRET=your-local-wdf-secret
```

**Dev Cluster** (_DEV suffix):
```bash
export GITHUB_TOOLS_APP_CLIENT_ID_DEV=your-dev-client-id
export GITHUB_TOOLS_APP_CLIENT_SECRET_DEV=your-dev-client-secret
export GITHUB_WDF_APP_CLIENT_ID_DEV=your-dev-wdf-id
export GITHUB_WDF_APP_CLIENT_SECRET_DEV=your-dev-wdf-secret
```

**Prod Cluster** (_PROD suffix):
```bash
export GITHUB_TOOLS_APP_CLIENT_ID_PROD=your-prod-client-id
export GITHUB_TOOLS_APP_CLIENT_SECRET_PROD=your-prod-client-secret
export GITHUB_WDF_APP_CLIENT_ID_PROD=your-prod-wdf-id
export GITHUB_WDF_APP_CLIENT_SECRET_PROD=your-prod-wdf-secret
```

> **Note**: Copy `env.example` to `.env` and fill in all environment-specific variables.

### Manual Build for Production
```bash
# Build optimized binary
make build-prod

# Or build Docker images manually
make docker-build-backend TAG=1.0.7
```

### Environment Setup
```bash
# Generate secure secrets
export JWT_SECRET=$(openssl rand -base64 32)
export DB_PASSWORD=$(openssl rand -base64 32)
export OAUTH_ENCRYPTION_KEY=$(openssl rand -base64 32)

# Set production environment
export ENVIRONMENT=production

# Configure all required variables (see env.example for full list)
export LDAP_HOST=ldap.example.com
export JIRA_DOMAIN=jira.example.com
export SONAR_HOST=https://sonarqube.example.com

# Restrict CORS origins
export ALLOWED_ORIGINS=https://your-frontend-domain.com
```

### Kubernetes Deployment

Deploy to Kubernetes using the Helm chart:

```bash
# Dev environment
cd charts/developer-portal-backend
./deploy.sh dev

# Prod environment
./deploy.sh prod
```

The deploy script will:
- Validate required environment variables
- Map environment-specific GitHub OAuth credentials
- Configure ingress hosts (dev: `dev.backend.*`, prod: `backend.*`)
- Deploy using Helm with all configurations

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and linting (`make test-precommit`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

### Development Guidelines
- Follow Go best practices and conventions
- Write tests for new features (aim for >80% coverage)
- Update documentation as needed
- Use meaningful commit messages
- Ensure code passes all checks (`make test-precommit`)

## 🐛 Troubleshooting

### Common Issues

**1. Database Connection Failed**
```bash
# Check if database is running
make db-up

# Verify connection
docker-compose -f docker/docker-compose.yml logs postgres
```

**2. JWT Secret Missing**
```bash
# For development
make run-dev  # Auto-generates secret

# For production
export JWT_SECRET=$(openssl rand -base64 32)
make run
```

**3. Authentication Issues**
- Check GitHub OAuth app configuration
- Verify client ID/secret environment variables
- Ensure redirect URLs match configuration

**4. Port Already in Use**
```bash
# Change port in .env
PORT=8080

# Or kill process using port
lsof -ti:7008 | xargs kill
```

**5. Migration Errors**
```bash
# Check migration files
ls internal/database/migrations/

# Force migration version if needed
make migrate-force VERSION=1
```

## 📚 Additional Resources

- [Gin Documentation](https://gin-gonic.com/docs/)
- [GORM Documentation](https://gorm.io/docs/)
- [golang-migrate](https://github.com/golang-migrate/migrate)
- [Backstage Authentication](https://backstage.io/docs/auth/)
- [Docker Compose](https://docs.docker.com/compose/)

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙋‍♂️ Support

For support and questions:
- Create an issue in the repository
- Check existing documentation
- Review troubleshooting section

---

**Happy Coding! 🚀**


## Test Structure

### File Naming Convention

- Test files: `*_test.go`
- Mock files: `*_mocks.go` (in `internal/mocks/`) 
- Factory files: `factories.go` (in `internal/testutils/`)

---

## Service Layer Testing

### Core Principles

1. **Use Mocks**: Mock all external dependencies (repositories, other services)
2. **Use Interfaces**: Depend on interfaces, not concrete implementations
3. **Use Factories**: Create test data using factory methods
4. **Test Suites**: Use testify/suite for organized test structure
5. **Package Suffix**: Use `package service_test` for proper encapsulation

example:  `internal/service/user_test.go`

## Repository Layer Testing

### Core Principles

1. **Use Real Database**: Tests run against actual PostgreSQL in Docker
2. **Use Factories**: Create test data using factory methods
3. **Use Test Suite**: Leverage BaseTestSuite for database management
4. **Integration Tag**: Mark tests with `//go:build integration`
5. **Clean Database**: Database is cleaned before/after each test

### Repository Test Template

```go
//go:build integration
// +build integration

package repository

import (
	"testing"
	
	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/testutils"
	
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// YourRepositoryTestSuite tests the YourRepository
type YourRepositoryTestSuite struct {
	suite.Suite
	baseTestSuite *testutils.BaseTestSuite
	repo          *YourRepository
	factories     *testutils.FactorySet
}

// SetupSuite runs once before all tests
func (suite *YourRepositoryTestSuite) SetupSuite() {
	// Initialize shared test database
	suite.baseTestSuite = testutils.SetupTestSuite(suite.T())
	
	// Initialize repository and factories
	suite.repo = NewYourRepository(suite.baseTestSuite.DB)
	suite.factories = testutils.NewFactorySet()
}

// TearDownSuite runs once after all tests
func (suite *YourRepositoryTestSuite) TearDownSuite() {
	suite.baseTestSuite.TeardownTestSuite()
}

// SetupTest runs before each test
func (suite *YourRepositoryTestSuite) SetupTest() {
	suite.baseTestSuite.SetupTest()
}

// TearDownTest runs after each test
func (suite *YourRepositoryTestSuite) TearDownTest() {
	suite.baseTestSuite.TearDownTest()
}

// Test methods
func (suite *YourRepositoryTestSuite) TestCreate() {
	// Arrange - create test entity using factory
	entity := suite.factories.User.Create()
	
	// Act - create in database
	err := suite.repo.Create(entity)
	
	// Assert
	suite.NoError(err)
	suite.NotEqual(uuid.Nil, entity.ID)
	suite.NotZero(entity.CreatedAt)
	suite.NotZero(entity.UpdatedAt)
}

// Run the test suite
func TestYourRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(YourRepositoryTestSuite))
}
```
example: `internal/repository/user_test.go`

## Test Utilities

### Factories

Factories provide a consistent way to create test data.

#### Available Factories

```go
factories := testutils.NewFactorySet()

// Organization
org := factories.Organization.Create()
org := factories.Organization.WithName("custom-org")

// User
user := factories.User.Create()
user := factories.User.WithEmail("test@example.com")
user := factories.User.WithRole(models.TeamRoleManager)
user := factories.User.WithTeam(teamID)

// Group
group := factories.Group.Create()
group := factories.Group.WithOrganization(orgID)
group := factories.Group.WithName("custom-group")

// Team
team := factories.Team.Create()
team := factories.Team.WithGroup(groupID)
team := factories.Team.WithOrganization(orgID)
team := factories.Team.WithName("custom-team")

// Project
project := factories.Project.Create()
project := factories.Project.WithName("custom-project")

// Component
component := factories.Component.Create()
component := factories.Component.WithName("custom-component")

// Landscape
landscape := factories.Landscape.Create()
landscape := factories.Landscape.WithName("custom-landscape")
```

### Test Suite

The BaseTestSuite provides database management for integration tests.

```go
// Initialize test suite
baseTestSuite := testutils.SetupTestSuite(t)

// Access database
db := baseTestSuite.DB

// Access config
config := baseTestSuite.Config

// Clean database (automatically called in SetupTest/TearDownTest)
baseTestSuite.CleanTestDB()

// Teardown (automatically called in TearDownSuite)
baseTestSuite.TeardownTestSuite()
```

### Using Custom Error Types

The project defines custom error types in `internal/errors/errors.go`. **Use these predefined error instances** in your tests instead of creating new error instances.


---

## Running Tests

### Run All Tests

```bash
# Run with coverage
make test-coverage-full
```

### Run with Coverage

```bash
# Generate coverage report
make test-coverage-full

# View coverage in browser
open coverage.html

# Check coverage for specific package
go test -cover ./internal/service/user_test.go
```

---

## Current Test Coverage Report

### API Middleware
| File | Coverage |
|------|----------|
| internal/api/middleware/cors.go | 0.0% |
| internal/api/middleware/logging.go | 0.0% |

### Authentication
| File | Coverage |
|------|----------|
| internal/auth/config.go | 21.8% |
| internal/auth/crypto.go | 59.7% |
| internal/auth/github.go | 26.1% |
| internal/auth/handlers.go | 15.1% |
| internal/auth/middleware.go | 39.7% |
| internal/auth/service.go | 40.6% |

### Configuration
| File | Coverage |
|------|----------|
| internal/config/config.go | 0.0% |

### Database
| File | Coverage |
|------|----------|
| internal/database/database.go | 75.3% |
| internal/database/init_data.go | 47.2% |

### Repository Layer
| File | Coverage |
|------|---------|
| internal/repository/component.go | 15.2%  |
| internal/repository/documentation.go | 0.0%  |
| internal/repository/group.go | 66.7% |
| internal/repository/landscape.go | 19.4% |
| internal/repository/link.go | 61.1% |
| internal/repository/organization.go | 35.1% |
| internal/repository/project.go | 25.3% |
| internal/repository/team.go | 20.6% |
| internal/repository/user.go | 1.7% |

### Service Layer
| File | Coverage |
|------|----------|
| internal/service/aicore_stream.go | 0.0% |
| internal/service/alerts.go | 0.0% |
| internal/service/component.go | 0.0% |
| internal/service/project.go | 0.0% |

---

## AI Testing Prompts

### 1. Service Layer Testing Prompt (Unit Tests with Mocks)

I want you to act as a Senior Go Backend Developer.

Go over the service file: [service_file.go] - we are going to write unit tests for it, so analyze the file: functions, types, dependencies, etc.

Follow these guidelines:

**1. Understand the Codebase**: Analyze the Go service file thoroughly, step by step. Identify all dependencies (repositories, other services), business logic, and error handling. If an existing test file is provided: analyze it fully, identify gaps and missing scenarios.

**2. Testing Framework**: Use testify/suite for test organization, testify/assert for assertions, and gomock for mocking.

**3. Test Structure** - Reference: `internal/service/user_test.go`
- Use `package service_test`
- Mock ALL external dependencies (repositories, services) using gomock
- Use testify/suite structure with SetupTest/TearDownTest
- Test business logic in isolation

**4. Project Conventions**
- Interfaces: `internal/repository/interfaces.go`, `internal/service/interfaces.go`
- Mocks: `internal/mocks/*` (use gomock expectations)
- Factories: `internal/testutils/factories.go` (for creating test data)
- Errors: `internal/errors/errors.go` (use predefined errors like `ErrUserNotFound`, `ErrUserExists`)

**5. Test Design**: Follow AAA pattern (Arrange → Act → Assert). Test Naming: `Test<MethodName>_<Scenario>`.

**6. Your Objective**: Create a robust, complete test suite that:
- Achieves >80% code coverage
- Tests happy paths, error scenarios (validation, not found, already exists), and edge cases
- Tests edge cases to catch potential bugs that might not be apparent in regular use
- Uses factories for test data and predefined errors
- Focuses on one functionality per test, keeps tests isolated
- Writes complete test cases (not skeletons)

---

### 2. Repository Layer Testing Prompt (Integration Tests with Database)

I want you to act as a Senior Go Backend Developer.

Go over the repository file: [repository_file.go] - we are going to write integration tests for it, so analyze the file: functions, types, database operations, etc.

 Follow these guidelines:

**1. Understand the Codebase**: Analyze the Go repository file thoroughly, step by step. Identify all database operations (CRUD, queries), relationships, and error handling. If an existing test file is provided: analyze it fully, identify gaps and missing scenarios.

**2. Testing Framework**: Use testify/suite for test organization, testify/assert for assertions, and real PostgreSQL database.

**3. Test Structure** - Reference: `internal/repository/user_test.go`
- Use `//go:build integration` tag at the top
- Use `testutils.BaseTestSuite` for database management
- Database is automatically cleaned before/after each test
- Test against real PostgreSQL database (no mocks)

**4. Project Conventions**
- Test Suite: `internal/testutils/suite.go` (BaseTestSuite)
- Factories: `internal/testutils/factories.go` (for creating test entities)
- Errors: `internal/errors/errors.go` (use predefined errors, check with `gorm.ErrRecordNotFound`)

**5. Test Design**: Follow AAA pattern (Arrange → Act → Assert). Test Naming: `Test<MethodName>_<Scenario>`.

**6. Your Objective**: Create a robust, complete integration test suite that:
- Achieves >80% code coverage
- Tests happy paths, error scenarios (not found, constraint violations), and edge cases
- Tests edge cases to catch potential bugs that might not be apparent in regular use
- Uses factories to create test data in database
- Tests database relationships and constraints
- Focuses on one functionality per test, keeps tests isolated
- Writes complete test cases (not skeletons)

---

### 3. General Testing Prompt (Auth, Middleware, Utilities)

I want you to act as a Senior Go Backend Developer.

Go over the file: [file.go] - we are going to write tests for it, so analyze the file: functions, types, dependencies, etc.

 Follow these guidelines:

**1. Understand the Codebase**: Analyze the Go file thoroughly, step by step. Identify all functions, dependencies, and logic flows. If an existing test file is provided: analyze it fully, identify gaps and missing scenarios.

**2. Testing Framework**: Use testify/suite for test organization and testify/assert for assertions.

**3. Test Structure** - Reference: `internal/auth/middleware_test.go`, `internal/auth/crypto_test.go`
- Use appropriate test structure based on the code type
- Mock external dependencies when needed
- Use testify/suite if multiple related tests exist

**4. Project Conventions**
- Factories: `internal/testutils/factories.go` (for creating test data)
- Errors: `internal/errors/errors.go` (use predefined errors like `ErrAuthenticationRequired`, `ErrInvalidRefreshToken`)
- HTTP Testing: `internal/testutils/http.go` (if testing HTTP handlers)

**5. Test Design**: Follow AAA pattern (Arrange → Act → Assert). Test Naming: `Test<MethodName>_<Scenario>`.

**6. Your Objective**: Create a robust, complete test suite that:
- Achieves >80% code coverage
- Tests happy paths, error scenarios, and edge cases
- Tests edge cases to catch potential bugs that might not be apparent in regular use
- Uses factories for test data and predefined errors
- Focuses on one functionality per test, keeps tests isolated
- Writes complete test cases (not skeletons)

---

package handlers_test

import (
	"database/sql"
	apperrors "developer-portal-backend/internal/errors"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"developer-portal-backend/internal/api/handlers"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type HealthHandlerTestSuite struct {
	suite.Suite
	handler *handlers.HealthHandler
	db      *gorm.DB
	sqlDB   *sql.DB
	mock    sqlmock.Sqlmock
}

func (suite *HealthHandlerTestSuite) SetupTest() {
	gin.SetMode(gin.TestMode)

	// Create a mock database connection with ping monitoring enabled
	var err error
	suite.sqlDB, suite.mock, err = sqlmock.New(sqlmock.MonitorPingsOption(true))
	suite.Require().NoError(err)

	// Expect the initial ping from GORM during initialization
	suite.mock.ExpectPing()

	// Create GORM DB with the mock
	dialector := postgres.New(postgres.Config{
		Conn:       suite.sqlDB,
		DriverName: "postgres",
	})
	suite.db, err = gorm.Open(dialector, &gorm.Config{})
	suite.Require().NoError(err)

	suite.handler = handlers.NewHealthHandler(suite.db)
}

func (suite *HealthHandlerTestSuite) TearDownTest() {
	if suite.sqlDB != nil {
		suite.sqlDB.Close()
	}
}

func (suite *HealthHandlerTestSuite) newRouter() *gin.Engine {
	r := gin.New()
	r.GET("/health", suite.handler.Health)
	r.GET("/health/ready", suite.handler.Ready)
	r.GET("/health/live", suite.handler.Live)
	r.GET("/cis-public/proxy", suite.handler.ProxyComponentHealth)
	return r
}

/*************** Health ***************/

// verifies the health endpoint returns healthy status when database is accessible
func (suite *HealthHandlerTestSuite) TestHealth_Success() {
	router := suite.newRouter()

	// Mock successful ping
	suite.mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response handlers.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response.Status)
	assert.Equal(suite.T(), "1.0.0", response.Version)
	assert.Equal(suite.T(), "healthy", response.Services["database"])
	assert.NotZero(suite.T(), response.Timestamp)

	// Verify all expectations were met
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// verifies the health endpoint returns unhealthy status when database ping fails
func (suite *HealthHandlerTestSuite) TestHealth_DatabasePingFailure() {
	router := suite.newRouter()

	// Mock failed ping
	suite.mock.ExpectPing().WillReturnError(errors.New("connection refused"))

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)

	var response handlers.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "unhealthy", response.Status)
	assert.Equal(suite.T(), "1.0.0", response.Version)
	assert.Contains(suite.T(), response.Services["database"], "error:")
	assert.Contains(suite.T(), response.Services["database"], "connection refused")

	// Verify all expectations were met
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// verifies the health endpoint handles database connection retrieval errors
func (suite *HealthHandlerTestSuite) TestHealth_DatabaseConnectionError() {
	// Create a handler with a closed database to simulate DB() error
	closedSQLDB, closedMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	suite.Require().NoError(err)

	// Expect ping during GORM initialization
	closedMock.ExpectPing()

	dialector := postgres.New(postgres.Config{
		Conn:       closedSQLDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	suite.Require().NoError(err)

	closedSQLDB.Close() // Close after GORM initialization

	handler := handlers.NewHealthHandler(gormDB)

	router := gin.New()
	router.GET("/health", handler.Health)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)

	var response handlers.HealthResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "unhealthy", response.Status)
	assert.Contains(suite.T(), response.Services["database"], "error:")

	// Verify expectations
	assert.NoError(suite.T(), closedMock.ExpectationsWereMet())
}

/*************** Ready ***************/

// verifies the readiness endpoint returns ready status when database is accessible
func (suite *HealthHandlerTestSuite) TestReady_Success() {
	router := suite.newRouter()

	// Mock successful ping
	suite.mock.ExpectPing()

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["ready"])
	assert.NotNil(suite.T(), response["timestamp"])

	services := response["services"].(map[string]interface{})
	assert.Equal(suite.T(), "ready", services["database"])

	// Verify all expectations were met
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// verifies the readiness endpoint returns not ready when database ping fails
func (suite *HealthHandlerTestSuite) TestReady_DatabaseNotReady_PingFailure() {
	router := suite.newRouter()

	// Mock failed ping
	suite.mock.ExpectPing().WillReturnError(errors.New("database not ready"))

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), false, response["ready"])

	services := response["services"].(map[string]interface{})
	assert.Contains(suite.T(), services["database"], "not ready:")
	assert.Contains(suite.T(), services["database"], "database not ready")

	// Verify all expectations were met
	assert.NoError(suite.T(), suite.mock.ExpectationsWereMet())
}

// verifies the readiness endpoint handles database connection retrieval errors
func (suite *HealthHandlerTestSuite) TestReady_DatabaseConnectionError() {
	// Create a handler with a closed database
	closedSQLDB, closedMock, err := sqlmock.New(sqlmock.MonitorPingsOption(true))
	suite.Require().NoError(err)

	// Expect ping during GORM initialization
	closedMock.ExpectPing()

	dialector := postgres.New(postgres.Config{
		Conn:       closedSQLDB,
		DriverName: "postgres",
	})
	gormDB, err := gorm.Open(dialector, &gorm.Config{})
	suite.Require().NoError(err)

	closedSQLDB.Close() // Close after GORM initialization

	handler := handlers.NewHealthHandler(gormDB)

	router := gin.New()
	router.GET("/health/ready", handler.Ready)

	req := httptest.NewRequest(http.MethodGet, "/health/ready", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusServiceUnavailable, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), false, response["ready"])

	services := response["services"].(map[string]interface{})
	assert.Contains(suite.T(), services["database"], "not ready:")

	// Verify expectations
	assert.NoError(suite.T(), closedMock.ExpectationsWereMet())
}

/*************** Live ***************/

// verifies the liveness endpoint returns alive status with valid timestamp
func (suite *HealthHandlerTestSuite) TestLive_Success() {
	router := suite.newRouter()

	req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), true, response["alive"])
	assert.NotNil(suite.T(), response["timestamp"])

	// Parse timestamp to ensure it's valid
	timestampStr, ok := response["timestamp"].(string)
	assert.True(suite.T(), ok)
	_, err = time.Parse(time.RFC3339, timestampStr)
	assert.NoError(suite.T(), err)
}

// verifies the liveness endpoint consistently returns OK status
func (suite *HealthHandlerTestSuite) TestLive_AlwaysReturnsOK() {
	// Live endpoint should always return 200 OK as long as the service can respond
	router := suite.newRouter()

	// Make multiple requests to ensure consistency
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest(http.MethodGet, "/health/live", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err)
		assert.Equal(suite.T(), true, response["alive"])
	}
}

/*************** ProxyComponentHealth ***************/

// verifies the proxy endpoint successfully forwards health check to external component
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_Success() {
	router := suite.newRouter()

	// Create a test server that returns a valid JSON response
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":  "healthy",
			"version": "1.2.3",
		})
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "healthy", response["status"])
	assert.Equal(suite.T(), "1.2.3", response["version"])
	assert.NotNil(suite.T(), response["responseTime"])
	assert.Equal(suite.T(), float64(200), response["statusCode"])
	assert.Equal(suite.T(), true, response["componentSuccess"])
}

// verifies the proxy endpoint returns error when URL parameter is missing
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_MissingURL() {
	router := suite.newRouter()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadRequest, w.Code)

	var response handlers.ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response.Error, apperrors.NewMissingQueryParam("url").Error())
}

// verifies the proxy endpoint handles invalid or unreachable URLs
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_InvalidURL() {
	router := suite.newRouter()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url=http://invalid-host-that-does-not-exist-12345.com", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "Failed to fetch from component endpoint")
	assert.NotNil(suite.T(), response["responseTime"])
}

// verifies the proxy endpoint handles error responses from external component
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_ComponentReturnsError() {
	router := suite.newRouter()

	// Create a test server that returns an error status
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "internal server error",
		})
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should still return 200 to frontend, but componentSuccess should be false
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), "internal server error", response["error"])
	assert.Equal(suite.T(), float64(500), response["statusCode"])
	assert.Equal(suite.T(), false, response["componentSuccess"])
	assert.NotNil(suite.T(), response["responseTime"])
}

// verifies the proxy endpoint handles invalid JSON responses from external component
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_InvalidJSONResponse() {
	router := suite.newRouter()

	// Create a test server that returns invalid JSON
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("not valid json"))
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	// The error field contains the error object, not a string
	assert.NotNil(suite.T(), response["error"])
	assert.Equal(suite.T(), float64(200), response["statusCode"])
	assert.Equal(suite.T(), false, response["componentSuccess"])
	assert.NotNil(suite.T(), response["responseTime"])
}

// verifies the proxy endpoint handles 404 responses from external component
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_ComponentReturns404() {
	router := suite.newRouter()

	// Create a test server that returns 404
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": "not found",
		})
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Should return 200 to frontend with componentSuccess false
	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(404), response["statusCode"])
	assert.Equal(suite.T(), false, response["componentSuccess"])
}

// verifies the proxy endpoint handles various 2xx success status codes
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_ComponentReturns2xxSuccess() {
	router := suite.newRouter()

	// Test various 2xx status codes - skip 204 as it has no content
	testCases := []int{200, 201}

	for _, statusCode := range testCases {
		testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(statusCode)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status": "ok",
			})
		}))

		req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(suite.T(), http.StatusOK, w.Code, "Failed for status code %d", statusCode)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(suite.T(), err, "Failed to unmarshal response for status code %d", statusCode)
		assert.Equal(suite.T(), float64(statusCode), response["statusCode"], "Wrong status code in response for %d", statusCode)
		assert.Equal(suite.T(), true, response["componentSuccess"], "componentSuccess should be true for status code %d", statusCode)

		testServer.Close()
	}
}

// verifies the proxy endpoint handles timeout when external component is slow
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_Timeout() {
	router := suite.newRouter()

	// Create a test server that delays response beyond timeout
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(12 * time.Second) // Longer than the 10 second timeout
		w.WriteHeader(http.StatusOK)
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusBadGateway, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Contains(suite.T(), response["error"], "Failed to fetch from component endpoint")
	assert.NotNil(suite.T(), response["responseTime"])
}

// verifies the proxy endpoint handles empty JSON responses from external component
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_EmptyResponse() {
	router := suite.newRouter()

	// Create a test server that returns empty JSON object
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{})
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)
	assert.Equal(suite.T(), float64(200), response["statusCode"])
	assert.Equal(suite.T(), true, response["componentSuccess"])
	assert.NotNil(suite.T(), response["responseTime"])
}

// verifies the proxy endpoint accurately tracks response time
func (suite *HealthHandlerTestSuite) TestProxyComponentHealth_ResponseTimeTracking() {
	router := suite.newRouter()

	// Create a test server with a small delay
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "ok",
		})
	}))
	defer testServer.Close()

	req := httptest.NewRequest(http.MethodGet, "/cis-public/proxy?url="+testServer.URL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(suite.T(), http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(suite.T(), err)

	// Response time should be at least 100ms
	responseTime, ok := response["responseTime"].(float64)
	assert.True(suite.T(), ok)
	assert.GreaterOrEqual(suite.T(), responseTime, float64(100))
}

// TestHealthHandlerTestSuite runs the health handler test suite
func TestHealthHandlerTestSuite(t *testing.T) {
	suite.Run(t, new(HealthHandlerTestSuite))
}

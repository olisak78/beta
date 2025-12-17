package handlers

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"developer-portal-backend/internal/database/models"
	"developer-portal-backend/internal/mocks"
	"developer-portal-backend/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

 // stubProjectServiceHandler implements ProjectServiceInterface for handler tests
type stubProjectServiceHandler struct {
	tmpl  string
	regex string
}

func (s stubProjectServiceHandler) GetAllProjects() ([]models.Project, error) { return nil, nil }
func (s stubProjectServiceHandler) GetHealthMetadata(projectID uuid.UUID) (string, string, error) {
	return s.tmpl, s.regex, nil
}

func setupGin() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestComponentHealth_CIS20_MatchAndNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	// Success regex for cis20: '"status":"UP"'
	// We will set template to the httptest server URL to avoid real network calls.
	matchBody := `{"status":"UP","details":{"db":"ok"}}`
	noMatchBody := `{"status":"DOWN","details":{"db":"down"}}`

	// Create a test server to simulate the health endpoint
	var respBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, respBody)
	}))
	defer ts.Close()

	projectSvc := stubProjectServiceHandler{
		tmpl:  ts.URL + "/health", // final URL: http://127.0.0.1:xxxx/health
		regex: `"status":"UP"`,
	}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	metadata := json.RawMessage(`{"health":true,"subdomain":"api"}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "accounts-service",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil).Times(2)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "example.com"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil).Times(2)

	h := NewComponentHandler(componentSvc, landscapeSvc, nil, projectSvc)

	router := setupGin()
	router.GET("/components/health", h.ComponentHealth)

	// Case 1: Body matches regex -> healthy = true
	respBody = matchBody
	req1 := httptest.NewRequest(http.MethodGet, "/components/health?component-id="+componentID.String()+"&landscape-id="+landscapeID.String(), nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	var payload1 map[string]any
	err := json.Unmarshal(w1.Body.Bytes(), &payload1)
	assert.NoError(t, err)
	assert.Equal(t, true, payload1["healthy"])
	assert.Equal(t, matchBody, payload1["details"])
	assert.Equal(t, float64(http.StatusOK), payload1["statusCode"])
	assert.Equal(t, ts.URL+"/health", payload1["healthURL"])

	// Case 2: Body does not match regex -> healthy = false
	respBody = noMatchBody
	req2 := httptest.NewRequest(http.MethodGet, "/components/health?component-id="+componentID.String()+"&landscape-id="+landscapeID.String(), nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var payload2 map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &payload2)
	assert.NoError(t, err)
	assert.Equal(t, false, payload2["healthy"])
	assert.Equal(t, noMatchBody, payload2["details"])
	assert.Equal(t, float64(http.StatusOK), payload2["statusCode"])
	assert.Equal(t, ts.URL+"/health", payload2["healthURL"])
}

func TestComponentHealth_USRV_MatchAndNoMatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	componentSvc := mocks.NewMockComponentServiceInterface(ctrl)
	landscapeSvc := mocks.NewMockLandscapeServiceInterface(ctrl)

	// Success regex for usrv: '(?s)^(UP|Overall status: UP.*)$'
	// We will template to server URL + {health_suffix} and provide suffix via component metadata
	matchBody := "Overall status: UP\nEverything looks good"
	noMatchBody := "Overall status: DOWN\nSomething is wrong"

	var respBody string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, respBody)
	}))
	defer ts.Close()

	projectSvc := stubProjectServiceHandler{
		tmpl:  ts.URL + "{health_suffix}",          // final URL: http://127.0.0.1:xxxx/<suffix>
		regex: `(?s)^(UP|Overall status: UP.*)$`, // from projects.yaml for usrv
	}

	componentID := uuid.New()
	projectID := uuid.New()
	landscapeID := uuid.New()

	// Provide health flag and suffix as per usrv components convention
	metadata := json.RawMessage(`{"health":true,"health_suffix":"/unified-account-health"}`)
	component := &models.Component{
		BaseModel: models.BaseModel{
			ID:       componentID,
			Name:     "unified-account",
			Metadata: metadata,
		},
		ProjectID: projectID,
	}
	componentSvc.EXPECT().GetByID(componentID).Return(component, nil).Times(2)

	landscape := &service.LandscapeResponse{ID: landscapeID, Domain: "atom.jaas-gcp.cloud.sap.corp"}
	landscapeSvc.EXPECT().GetLandscapeByID(landscapeID).Return(landscape, nil).Times(2)

	h := NewComponentHandler(componentSvc, landscapeSvc, nil, projectSvc)

	router := setupGin()
	router.GET("/components/health", h.ComponentHealth)

	expectedHealthURL := ts.URL + "/unified-account-health"

	// Case 1: Body matches regex -> healthy = true
	respBody = matchBody
	req1 := httptest.NewRequest(http.MethodGet, "/components/health?component-id="+componentID.String()+"&landscape-id="+landscapeID.String(), nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)
	var payload1 map[string]any
	err := json.Unmarshal(w1.Body.Bytes(), &payload1)
	assert.NoError(t, err)
	assert.Equal(t, true, payload1["healthy"])
	assert.Equal(t, matchBody, payload1["details"])
	assert.Equal(t, float64(http.StatusOK), payload1["statusCode"])
	assert.Equal(t, expectedHealthURL, payload1["healthURL"])

	// Case 2: Body does not match regex -> healthy = false
	respBody = noMatchBody
	req2 := httptest.NewRequest(http.MethodGet, "/components/health?component-id="+componentID.String()+"&landscape-id="+landscapeID.String(), nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
	var payload2 map[string]any
	err = json.Unmarshal(w2.Body.Bytes(), &payload2)
	assert.NoError(t, err)
	assert.Equal(t, false, payload2["healthy"])
	assert.Equal(t, noMatchBody, payload2["details"])
	assert.Equal(t, float64(http.StatusOK), payload2["statusCode"])
	assert.Equal(t, expectedHealthURL, payload2["healthURL"])
}

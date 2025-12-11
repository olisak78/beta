package service

import (
	"bytes"
	"errors"
	"io"
	"time"
	"net/http"
	"testing"

	"developer-portal-backend/internal/cache"
	"developer-portal-backend/internal/config"

	"github.com/stretchr/testify/assert"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func jsonResponse(status int, body string) *http.Response {
	resp := &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(bytes.NewBufferString(body)),
		Header:     make(http.Header),
	}
	resp.Header.Set("Content-Type", "application/json")
	return resp
}

func baseSonarCfg() *config.Config {
	return &config.Config{
		SonarHost:  "sonar.example.com",
		SonarToken: "token-123",
	}
}

func newSonarWithTransport(cfg *config.Config, rt roundTripFunc) *SonarService {
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	s := NewSonarService(cfg, mockCache)
	s.httpClient = &http.Client{Transport: rt}
	return s
}

func TestSonar_GetComponentMeasures_Success(t *testing.T) {
	cfg := baseSonarCfg()
	const projectKey = "my-project"

	rt := func(req *http.Request) (*http.Response, error) {
		// Common header assertions for all Sonar calls
		assert.Equal(t, "Bearer "+cfg.SonarToken, req.Header.Get("Authorization"))
		assert.Equal(t, "application/json", req.Header.Get("Accept"))

		switch req.URL.Path {
		case "/api/measures/component":
			// Verify query params
			q := req.URL.Query()
			assert.Equal(t, projectKey, q.Get("component"))
			assert.Equal(t, "coverage,vulnerabilities,code_smells", q.Get("metricKeys"))

			return jsonResponse(200, `{
				"component": {
					"measures": [
						{"metric":"coverage","value":"85.3","bestValue":false},
						{"metric":"vulnerabilities","value":"0","bestValue":true},
						{"metric":"code_smells","value":"12","bestValue":false}
					]
				}
			}`), nil

		case "/api/qualitygates/project_status":
			// Verify query params
			q := req.URL.Query()
			assert.Equal(t, projectKey, q.Get("projectKey"))

			return jsonResponse(200, `{
				"projectStatus": { "status": "OK" }
			}`), nil
		default:
			return jsonResponse(404, `{"message":"not found"}`), nil
		}
	}

	svc := newSonarWithTransport(cfg, rt)
	res, err := svc.GetComponentMeasures(projectKey)
	assert.NoError(t, err)
	if assert.NotNil(t, res) {
		assert.Equal(t, "OK", res.Status)
		assert.Len(t, res.Measures, 3)
		// Spot check measures presence
		found := map[string]bool{}
		for _, m := range res.Measures {
			found[m.Metric] = true
		}
		assert.True(t, found["coverage"])
		assert.True(t, found["vulnerabilities"])
		assert.True(t, found["code_smells"])
	}
}

func TestSonar_GetComponentMeasures_ConfigMissing(t *testing.T) {
	cfg := &config.Config{
		SonarHost: "", // missing
		SonarToken: "x",
	}
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	svc := NewSonarService(cfg, mockCache)
	res, err := svc.GetComponentMeasures("proj")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "sonar configuration missing")
}

func TestSonar_GetComponentMeasures_ProjectKeyMissing(t *testing.T) {
	cfg := baseSonarCfg()
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	svc := NewSonarService(cfg, mockCache)
	res, err := svc.GetComponentMeasures("")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "project key (component) is required")
}

func TestSonar_GetComponentMeasures_InvalidHostURL(t *testing.T) {
	cfg := baseSonarCfg()
	// Invalid host with space to trigger url.Parse error
	cfg.SonarHost = "bad host with space"
	mockCache := cache.NewInMemoryCache(5*time.Minute, 10*time.Minute)
	svc := NewSonarService(cfg, mockCache)
	res, err := svc.GetComponentMeasures("proj")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "invalid sonar host URL")
}

func TestSonar_GetComponentMeasures_MeasuresNon2xx(t *testing.T) {
	cfg := baseSonarCfg()

	rt := func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/measures/component":
			return jsonResponse(500, `oops`), nil
		case "/api/qualitygates/project_status":
			return jsonResponse(200, `{"projectStatus":{"status":"OK"}}`), nil
		default:
			return jsonResponse(404, `{"message":"not found"}`), nil
		}
	}

	svc := newSonarWithTransport(cfg, rt)
	res, err := svc.GetComponentMeasures("proj")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to fetch sonar measures")
	assert.Contains(t, err.Error(), "status=500")
}

func TestSonar_GetComponentMeasures_QualityGateNetworkError(t *testing.T) {
	cfg := baseSonarCfg()

	rt := func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/measures/component":
			return jsonResponse(200, `{"component":{"measures":[{"metric":"coverage","value":"80","bestValue":false}]}}`), nil
		case "/api/qualitygates/project_status":
			return nil, errors.New("network down")
		default:
			return jsonResponse(404, `{"message":"not found"}`), nil
		}
	}

	svc := newSonarWithTransport(cfg, rt)
	res, err := svc.GetComponentMeasures("proj")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to fetch sonar quality gate status")
}

func TestSonar_GetComponentMeasures_MeasuresDecodeError(t *testing.T) {
	cfg := baseSonarCfg()

	rt := func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/api/measures/component":
			// invalid JSON to trigger decode error
			return jsonResponse(200, `{"component":{"measures":[{"metric": }]}}`), nil
		case "/api/qualitygates/project_status":
			return jsonResponse(200, `{"projectStatus":{"status":"ERROR"}}`), nil
		default:
			return jsonResponse(404, `{"message":"not found"}`), nil
		}
	}

	svc := newSonarWithTransport(cfg, rt)
	res, err := svc.GetComponentMeasures("proj")
	assert.Error(t, err)
	assert.Nil(t, res)
	assert.Contains(t, err.Error(), "failed to fetch sonar measures")
	assert.Contains(t, err.Error(), "failed to decode sonar response")
}
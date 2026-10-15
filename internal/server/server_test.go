package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite"
	"github.com/the127/frogly/internal/server"
)

const (
	validToken   = "valid-token"
	otherToken   = "token-of-someone-else"
	outageToken  = "token-during-an-outage"
	refusedToken = "forged"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, accessToken string) (authn.Person, error) {
	switch accessToken {
	case validToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "Karo"}, nil

	case otherToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "ana-id", Name: "Ana"}, nil

	case outageToken:
		return authn.Person{}, errors.New("fetching the issuer's keys: connection refused")

	default:
		return authn.Person{}, errors.Join(authn.ErrUnauthenticated, errors.New("not the valid token"))
	}
}

func newServer(t *testing.T, log *bytes.Buffer) http.Handler {
	t.Helper()

	database, err := sqlite.Open(filepath.Join(t.TempDir(), "frogly.db"))
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, database.Close())
	})

	require.NoError(t, database.Migrate(t.Context()))

	dependencies := ioc.NewDependencyCollection()
	ioc.RegisterScoped(dependencies, func(*ioc.DependencyProvider) db.Context {
		return database.NewContext()
	})

	return server.New(
		config.OIDC{Issuer: "http://issuer.example", ClientID: "frogly"},
		fakeVerifier{},
		slog.New(slog.NewJSONHandler(log, nil)),
		frontend,
		dependencies.BuildProvider(),
	)
}

func projects(t *testing.T, handler http.Handler, method string, token string, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, "/api/projects", strings.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

var frontend = fstest.MapFS{
	"index.html":           {Data: []byte("<!doctype html><title>frogly</title>")},
	"assets/index-a1.js":   {Data: []byte("console.log('frogly')")},
	"assets/index-a1.css":  {Data: []byte("body{}")},
	"assets/fonts/a.woff2": {Data: []byte("font")},
}

func request(t *testing.T, handler http.Handler, method string, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, path, nil)
	for name, values := range header {
		request.Header[name] = values
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
}

func get(t *testing.T, handler http.Handler, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	return request(t, handler, http.MethodGet, path, header)
}

func TestConfigTellsTheFrontendWhereToLogIn(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/api/config", nil)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"issuer": "http://issuer.example", "clientId": "frogly"}`, response.Body.String())
}

func TestMeIsUnavailableWhenTheTokenCantBeChecked(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + outageToken}})

	// assert
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.Empty(t, response.Header().Get("WWW-Authenticate"))
	assert.JSONEq(t, `{"type": "about:blank", "title": "Service Unavailable", "status": 503}`, response.Body.String())
}

func TestMeRefusesRequestsWithoutAValidToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		wantChallenge string
	}{
		{name: "no token", authorization: "", wantChallenge: `Bearer`},
		{name: "another scheme", authorization: "Basic a2Fybzpmcm9nbHk=", wantChallenge: `Bearer`},
		{name: "empty token", authorization: "Bearer ", wantChallenge: `Bearer`},
		{name: "invalid token", authorization: "Bearer " + refusedToken, wantChallenge: `Bearer error="invalid_token"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			header := http.Header{}
			if test.authorization != "" {
				header.Set("Authorization", test.authorization)
			}

			// act
			response := get(t, handler, "/api/me", header)

			// assert
			assert.Equal(t, http.StatusUnauthorized, response.Code)
			assert.Equal(t, test.wantChallenge, response.Header().Get("WWW-Authenticate"))
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
			assert.JSONEq(t, `{"type": "about:blank", "title": "Unauthorized", "status": 401}`, response.Body.String())
		})
	}
}

func TestUnknownAPIPathsAreNotFound(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/api/nothing-here", nil)

	// assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String())
}

func TestEveryRequestIsLoggedWithItsRequestID(t *testing.T) {
	// arrange
	var log bytes.Buffer

	handler := newServer(t, &log)

	// act
	response := get(t, handler, "/api/config", nil)

	// assert
	requestID := response.Header().Get("X-Request-Id")
	require.NotEmpty(t, requestID)

	var line struct {
		Msg       string `json:"msg"`
		RequestID string `json:"request_id"`
		Method    string `json:"method"`
		Path      string `json:"path"`
		Status    int    `json:"status"`
	}

	require.NoError(t, json.Unmarshal(log.Bytes(), &line))
	assert.Equal(t, "request", line.Msg)
	assert.Equal(t, requestID, line.RequestID)
	assert.Equal(t, http.MethodGet, line.Method)
	assert.Equal(t, "/api/config", line.Path)
	assert.Equal(t, http.StatusOK, line.Status)
}

func TestTheLogHasTheStatusButNotTheToken(t *testing.T) {
	// arrange
	var log bytes.Buffer

	handler := newServer(t, &log)

	// act
	get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + refusedToken}})

	// assert
	assert.Contains(t, log.String(), `"status":401`)
	assert.NotContains(t, log.String(), refusedToken)
}

func TestTheServerChoosesTheRequestID(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	sent := http.Header{"X-Request-Id": {"chosen-by-the-client"}}

	// act
	first := get(t, handler, "/api/config", sent)
	second := get(t, handler, "/api/config", sent)

	// assert
	assert.NotEqual(t, "chosen-by-the-client", first.Header().Get("X-Request-Id"))
	assert.NotEqual(t, first.Header().Get("X-Request-Id"), second.Header().Get("X-Request-Id"))
}

func TestServesTheFrontend(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantBody         string
		wantCacheControl string
	}{
		{
			name:             "the start page",
			path:             "/",
			wantBody:         "<!doctype html><title>frogly</title>",
			wantCacheControl: "no-cache",
		},
		{
			name:             "a page only the frontend knows, like the login callback",
			path:             "/callback",
			wantBody:         "<!doctype html><title>frogly</title>",
			wantCacheControl: "no-cache",
		},
		{
			name:             "a built asset",
			path:             "/assets/index-a1.js",
			wantBody:         "console.log('frogly')",
			wantCacheControl: "public, max-age=31536000, immutable",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := get(t, handler, test.path, nil)

			// assert
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, test.wantBody, response.Body.String())
			assert.Equal(t, test.wantCacheControl, response.Header().Get("Cache-Control"))
		})
	}
}

func TestAMissingAssetIsNotFound(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/assets/index-gone.js", nil)

	// assert
	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestNoAssetsDirectoryIsListed(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		wantCode int
	}{
		{name: "the assets directory itself, which is a page of the frontend", path: "/assets/", wantCode: http.StatusOK},
		{name: "a directory inside it", path: "/assets/fonts/", wantCode: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := get(t, handler, test.path, nil)

			// assert
			assert.Equal(t, test.wantCode, response.Code)
			assert.NotContains(t, response.Body.String(), "a.woff2")
			assert.NotContains(t, response.Body.String(), "index-a1.js")
		})
	}
}

func TestAPathLeavingTheAssetsServesTheFrontendInstead(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/assets/../../etc/passwd", nil)

	// assert
	assert.NotContains(t, response.Body.String(), "root:")
}

func TestTheFrontendAnswersHead(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := request(t, handler, http.MethodHead, "/", nil)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Empty(t, response.Body.String())
}

func TestMeGreetsThePersonByName(t *testing.T) {
	for _, scheme := range []string{"Bearer", "bearer"} {
		t.Run(scheme, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := get(t, handler, "/api/me", http.Header{"Authorization": {scheme + " " + validToken}})

			// assert
			assert.Equal(t, http.StatusOK, response.Code)
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.JSONEq(t, `{"name": "Karo"}`, response.Body.String())
		})
	}
}

func TestOnlyReadsAreAllowed(t *testing.T) {
	for _, path := range []string{"/api/config", "/api/me", "/"} {
		t.Run(path, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := request(t, handler, http.MethodPost, path, nil)

			// assert
			assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
			assert.Equal(t, "GET, HEAD", response.Header().Get("Allow"))
		})
	}
}

func TestProjectsStartEmpty(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := projects(t, handler, http.MethodGet, validToken, "")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
}

func TestACreatedProjectIsListed(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	created := projects(t, handler, http.MethodPost, validToken, `{"name": "  Garden "}`)

	// assert
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "Garden", project.Name)

	listed := projects(t, handler, http.MethodGet, validToken, "")
	assert.JSONEq(t, `[{"id": "`+project.ID+`", "name": "Garden"}]`, listed.Body.String())
}

func TestOthersDontSeeYourProjects(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Garden"}`)
	require.Equal(t, http.StatusCreated, created.Code)

	// act
	response := projects(t, handler, http.MethodGet, otherToken, "")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
}

func TestAProjectFroglyCantUseIsABadRequest(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantDetail string
	}{
		{name: "no name", body: `{"name": "   "}`, wantDetail: "a project needs a name"},
		{name: "not JSON", body: `Garden`, wantDetail: "the body isn't a project in JSON"},
		{name: "an unknown field", body: `{"name": "Garden", "colour": "green"}`, wantDetail: "the body isn't a project in JSON"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := projects(t, handler, http.MethodPost, validToken, test.body)

			// assert
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"))

			var problem struct {
				Detail string `json:"detail"`
			}

			require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
			assert.Contains(t, problem.Detail, test.wantDetail)
		})
	}
}

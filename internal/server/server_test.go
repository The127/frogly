package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"testing/fstest"
	"testing/iotest"
	"time"

	"github.com/The127/ioc"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/db/sqlite/sqlitetest"
	"github.com/the127/frogly/internal/repositories"
	"github.com/the127/frogly/internal/server"
)

const (
	validToken   = "valid-token"
	otherToken   = "token-of-someone-else"
	renamedToken = "token-after-a-new-name" //nolint:gosec // a made-up token the fake verifier knows
	outageToken  = "token-during-an-outage"
	refusedToken = "forged"
	noRoleToken  = "token-without-froglys-role"

	userRole = "frogly:frogly-user"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, accessToken string) (authn.Person, error) {
	switch accessToken {
	case validToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "Karo", Roles: []string{userRole}}, nil

	case renamedToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "Karolin", Roles: []string{userRole}}, nil

	case otherToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "ana-id", Name: "Ana", Roles: []string{userRole}}, nil

	case noRoleToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "lea-id", Name: "Lea", Roles: []string{"frogly:admin"}}, nil

	case outageToken:
		return authn.Person{}, errors.New("fetching the issuer's keys: connection refused")

	default:
		return authn.Person{}, errors.Join(authn.ErrUnauthenticated, errors.New("not the valid token"))
	}
}

func newServer(t *testing.T, log *bytes.Buffer) http.Handler {
	t.Helper()

	handler, _ := newServerWithDatabase(t, log)

	return handler
}

func newServerWithDatabase(t *testing.T, log *bytes.Buffer) (http.Handler, db.Database) {
	t.Helper()

	return newServerWithUnits(t, log, func(unit db.Context) db.Context { return unit })
}

// newServerWithUnits lets a test change the unit of work each request gets
func newServerWithUnits(t *testing.T, log *bytes.Buffer, wrap func(db.Context) db.Context) (http.Handler, db.Database) {
	t.Helper()

	database := sqlitetest.Open(t)
	dependencies := ioc.NewDependencyCollection()
	ioc.RegisterSingleton(dependencies, func(*ioc.DependencyProvider) db.Database {
		return database
	})
	ioc.RegisterScoped(dependencies, func(services *ioc.DependencyProvider) db.Context {
		return wrap(ioc.GetDependency[db.Database](services).NewContext())
	})

	return server.New(
		config.OIDC{
			Issuer:   "http://issuer.example",
			ClientID: "frogly",
			Roles:    config.Roles{Claim: "roles", User: userRole},
		},
		fakeVerifier{},
		slog.New(slog.NewJSONHandler(log, nil)),
		frontend,
		dependencies.BuildProvider(),
	), database
}

func projects(t *testing.T, handler http.Handler, method string, token string, body string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{"Content-Type": {"application/json"}}
	if token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	return requestWithBody(t, handler, method, "/api/projects", header, strings.NewReader(body))
}

var frontend = fstest.MapFS{
	"index.html":           {Data: []byte("<!doctype html><title>frogly</title>")},
	"assets/index-a1.js":   {Data: []byte("console.log('frogly')")},
	"assets/index-a1.css":  {Data: []byte("body{}")},
	"assets/fonts/a.woff2": {Data: []byte("font")},
}

func request(t *testing.T, handler http.Handler, method string, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	return requestWithBody(t, handler, method, path, header, nil)
}

func requestWithBody(t *testing.T, handler http.Handler, method string, path string, header http.Header, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), method, path, body)
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

func TestAPersonWithoutFroglysRoleIsForbidden(t *testing.T) {
	for _, path := range []string{"/api/me", "/api/projects"} {
		t.Run(path, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := get(t, handler, path, http.Header{"Authorization": {"Bearer " + noRoleToken}})

			// assert
			assert.Equal(t, http.StatusForbidden, response.Code)
			assert.Empty(t, response.Header().Get("WWW-Authenticate"))
			assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
			assert.JSONEq(t, `{"type": "about:blank", "title": "Forbidden", "status": 403}`, response.Body.String())
		})
	}
}

func TestCreatingAProjectWithoutFroglysRoleIsForbidden(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := projects(t, handler, http.MethodPost, noRoleToken, `{"name": "Garden", "visibility": "private", "joining": "closed"}`)

	// assert
	assert.Equal(t, http.StatusForbidden, response.Code)
}

func TestAPersonTurnedAwayIsLogged(t *testing.T) {
	// arrange
	var log bytes.Buffer
	handler := newServer(t, &log)

	// act
	get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + noRoleToken}})

	// assert
	type line struct {
		Msg     string `json:"msg"`
		Issuer  string `json:"issuer"`
		Subject string `json:"subject"`
	}

	var lines []line
	for _, raw := range bytes.Split(bytes.TrimSpace(log.Bytes()), []byte("\n")) {
		var parsed line
		require.NoError(t, json.Unmarshal(raw, &parsed))
		lines = append(lines, parsed)
	}

	assert.Contains(t, lines, line{Msg: "turned away a person without frogly's role", Issuer: "http://issuer.example", Subject: "lea-id"})
}

func TestAPersonWithoutFroglysRoleIsntStored(t *testing.T) {
	// arrange
	handler, database := newServerWithDatabase(t, &bytes.Buffer{})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + noRoleToken}})

	// assert
	assert.Equal(t, http.StatusForbidden, response.Code)

	identity, err := database.NewContext().Identities().ByLogin(t.Context(), "http://issuer.example", "lea-id")
	require.NoError(t, err)
	assert.Nil(t, identity)
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
	created := projects(t, handler, http.MethodPost, validToken, `{"name": "  Garden ", "visibility": "public", "joining": "closed"}`)

	// assert
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID         string `json:"id"`
		Name       string `json:"name"`
		Visibility string `json:"visibility"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))
	assert.NotEmpty(t, project.ID)
	assert.Equal(t, "Garden", project.Name)
	assert.Equal(t, "public", project.Visibility)

	listed := projects(t, handler, http.MethodGet, validToken, "")
	assert.JSONEq(t, `[{"id": "`+project.ID+`", "name": "Garden", "visibility": "public", "joining": "closed"}]`, listed.Body.String())
}

func TestOthersDontSeeYourProjects(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	// public, because a public project still belongs only in its members' list
	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Garden", "visibility": "public", "joining": "closed"}`)
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
		{name: "no name", body: `{"name": "   ", "visibility": "private"}`, wantDetail: "a project needs a name"},
		{name: "too long a name", body: `{"name": "` + strings.Repeat("蛙", 101) + `", "visibility": "private"}`, wantDetail: "a project's name has at most 100 characters"},
		{name: "no visibility", body: `{"name": "Garden"}`, wantDetail: "a project is public or private"},
		{name: "a public project without how to join", body: `{"name": "Garden", "visibility": "public"}`, wantDetail: "a project is open to join, takes requests or is closed"},
		{name: "an open private project", body: `{"name": "Garden", "visibility": "private", "joining": "open"}`, wantDetail: "only a public project can be open to join or take requests"},
		{name: "an unknown visibility", body: `{"name": "Garden", "visibility": "secret"}`, wantDetail: "a project is public or private"},
		{name: "not JSON", body: `Garden`, wantDetail: "the body isn't the JSON this request takes"},
		{name: "an unknown field", body: `{"name": "Garden", "visibility": "private", "colour": "green"}`, wantDetail: "the body isn't the JSON this request takes"},
		{name: "something after the project", body: `{"name": "Garden", "visibility": "private"} {"name": "Board"}`, wantDetail: "the body isn't the JSON this request takes"},
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
			assert.Equal(t, test.wantDetail, problem.Detail)
		})
	}
}

func TestAnOversizedProjectIsTooLarge(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	body := `{"name": "` + strings.Repeat("a", 10<<10) + `"}`

	// act
	response := projects(t, handler, http.MethodPost, validToken, body)

	// assert
	assert.Equal(t, http.StatusRequestEntityTooLarge, response.Code)
}

func TestCreatingAProjectNeedsAToken(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})

	// act
	response := projects(t, handler, http.MethodPost, "", `{"name": "Garden"}`)

	// assert
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestAMemberGetsTheProject(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := get(t, handler, "/api/projects/"+projectID, http.Header{"Authorization": {"Bearer " + validToken}})

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Garden", "visibility": "private", "joining": "closed"}`, response.Body.String())
}

func TestAProjectYouCantSeeIsNotFound(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	tests := []struct {
		name      string
		token     string
		projectID string
	}{
		{name: "not a member", token: otherToken, projectID: projectID},
		{name: "no such project", token: validToken, projectID: "01a14180-d957-7cc5-af29-7f1fe18d89bb"},
		{name: "not an ID", token: validToken, projectID: "garden"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// act
			response := get(t, handler, "/api/projects/"+test.projectID, http.Header{"Authorization": {"Bearer " + test.token}})

			// assert
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String())
		})
	}
}

func TestMeAnswersWithTheLatestName(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + validToken}})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + renamedToken}})

	// assert
	assert.JSONEq(t, `{"name": "Karolin"}`, response.Body.String())
}

func TestABodyThatCantBeReadIsLoggedAndHidden(t *testing.T) {
	// arrange
	log := &bytes.Buffer{}
	handler := newServer(t, log)
	header := http.Header{"Authorization": {"Bearer " + validToken}}
	body := iotest.ErrReader(errors.New("the connection broke"))

	// act
	response := requestWithBody(t, handler, http.MethodPost, "/api/projects", header, body)

	// assert
	assert.Equal(t, http.StatusInternalServerError, response.Code)
	assert.NotContains(t, response.Body.String(), "the connection broke")
	assert.Contains(t, log.String(), "the connection broke")
}

func changeVisibility(t *testing.T, handler http.Handler, token string, projectID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + token}}

	return requestWithBody(t, handler, http.MethodPut, "/api/projects/"+projectID+"/visibility", header, strings.NewReader(body))
}

func TestAMemberChangesTheVisibility(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := changeVisibility(t, handler, validToken, projectID, `{"visibility": "public"}`)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Garden", "visibility": "public", "joining": "closed"}`, response.Body.String())

	got := get(t, handler, "/api/projects/"+projectID, http.Header{"Authorization": {"Bearer " + validToken}})
	assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Garden", "visibility": "public", "joining": "closed"}`, got.Body.String())
}

func TestAVisibilityYouCantChangeIsNotFound(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		projectID func(string) string
	}{
		{name: "not a member", token: otherToken, projectID: func(id string) string { return id }},
		{name: "no such project", token: validToken, projectID: func(string) string { return "01a14180-d957-7cc5-af29-7f1fe18d89bb" }},
		{name: "not an ID", token: validToken, projectID: func(string) string { return "garden" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})
			projectID := createdProject(t, handler)

			// act
			response := changeVisibility(t, handler, test.token, test.projectID(projectID), `{"visibility": "public"}`)

			// assert
			assert.Equal(t, http.StatusNotFound, response.Code)

			got := get(t, handler, "/api/projects/"+projectID, http.Header{"Authorization": {"Bearer " + validToken}})
			assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Garden", "visibility": "private", "joining": "closed"}`, got.Body.String())
		})
	}
}

func TestAPublicProjectYouArentInIsNotFound(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProjectWith(t, handler, "public")

	// act
	response := get(t, handler, "/api/projects/"+projectID, http.Header{"Authorization": {"Bearer " + otherToken}})

	// assert
	assert.Equal(t, http.StatusNotFound, response.Code)
}

func TestAnUnknownVisibilityIsABadRequest(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := changeVisibility(t, handler, validToken, projectID, `{"visibility": "secret"}`)

	// assert
	assert.Equal(t, http.StatusBadRequest, response.Code)

	var problem struct {
		Detail string `json:"detail"`
	}

	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
	assert.Equal(t, "a project is public or private", problem.Detail)
}

func TestOthersSeeAPublicProjectByItsName(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProjectWith(t, handler, "public")
	createdProjectWith(t, handler, "private")

	// act
	response := get(t, handler, "/api/projects/public", http.Header{"Authorization": {"Bearer " + otherToken}})

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[{"id": "`+projectID+`", "name": "Garden", "joining": "closed", "asked": false}]`, response.Body.String())
}

func TestYourOwnPublicProjectIsntAmongTheOthers(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	createdProjectWith(t, handler, "public")

	// act
	response := get(t, handler, "/api/projects/public", http.Header{"Authorization": {"Bearer " + validToken}})

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
}

// racedUnit saves like a unit of work that lost to another request
type racedUnit struct {
	db.Context
}

func (racedUnit) SaveChanges(context.Context) error {
	return fmt.Errorf("saving: %w", repositories.ErrConcurrentModification)
}

func TestAChangeThatLostARaceIsAConflict(t *testing.T) {
	// arrange
	var log bytes.Buffer
	raced := false
	handler, _ := newServerWithUnits(t, &log, func(unit db.Context) db.Context {
		if raced {
			return racedUnit{unit}
		}

		return unit
	})
	projectID := createdProject(t, handler)
	raced = true
	log.Reset()

	// act
	response := changeVisibility(t, handler, validToken, projectID, `{"visibility": "public"}`)

	// assert
	assert.Equal(t, http.StatusConflict, response.Code)
	assert.JSONEq(t, `{"type": "about:blank", "title": "Conflict", "status": 409}`, response.Body.String())
	assert.NotContains(t, log.String(), `"level":"ERROR"`)
}

func searchPeople(t *testing.T, handler http.Handler, token string, projectID string, text string) *httptest.ResponseRecorder {
	t.Helper()

	return get(t, handler, "/api/projects/"+projectID+"/people?search="+url.QueryEscape(text), http.Header{"Authorization": {"Bearer " + token}})
}

func addMember(t *testing.T, handler http.Handler, token string, projectID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + token}}

	return requestWithBody(t, handler, http.MethodPost, "/api/projects/"+projectID+"/members", header, strings.NewReader(body))
}

// anaID logs Ana in once, which is how frogly gets to know her
func anaID(t *testing.T, handler http.Handler, projectID string) string {
	t.Helper()

	require.Equal(t, http.StatusOK, get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + otherToken}}).Code)

	var found []struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(searchPeople(t, handler, validToken, projectID, "ana").Body.Bytes(), &found))
	require.Len(t, found, 1)

	return found[0].ID
}

func TestAMemberFindsSomeoneWhoLoggedIn(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)
	id := anaID(t, handler, projectID)

	// act
	response := searchPeople(t, handler, validToken, projectID, "AN")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[{"id": "`+id+`", "name": "Ana", "email": ""}]`, response.Body.String())
}

func TestASearchLeavesOutWhoeverSearches(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)
	id := anaID(t, handler, projectID)

	// act, a matches both Karo and Ana
	response := searchPeople(t, handler, validToken, projectID, "a")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[{"id": "`+id+`", "name": "Ana", "email": ""}]`, response.Body.String())
}

func TestASearchFindingNobodyIsAnEmptyList(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := searchPeople(t, handler, validToken, projectID, "nobody")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
}

func TestAnEmptySearchIsABadRequest(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := searchPeople(t, handler, validToken, projectID, " ")

	// assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "type part of a name or an email")
}

func TestAMemberAddsSomeoneWhoLoggedIn(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)
	id := anaID(t, handler, projectID)

	// act
	response := addMember(t, handler, validToken, projectID, `{"userId": "`+id+`"}`)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id": "`+id+`", "name": "Ana", "email": ""}`, response.Body.String())

	listed := projects(t, handler, http.MethodGet, otherToken, "")
	assert.Contains(t, listed.Body.String(), projectID)
}

func TestAddingSomeoneFroglyDoesntKnowIsABadRequest(t *testing.T) {
	for _, userID := range []string{"01a14180-d957-7cc5-af29-7f1fe18d89bb", "ana"} {
		t.Run(userID, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})
			projectID := createdProject(t, handler)

			// act
			response := addMember(t, handler, validToken, projectID, `{"userId": "`+userID+`"}`)

			// assert
			assert.Equal(t, http.StatusBadRequest, response.Code)
			assert.Contains(t, response.Body.String(), "nobody with that ID uses frogly")
		})
	}
}

func TestSomeoneOutsideTheProjectNeitherSearchesNorAdds(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProjectWith(t, handler, "public")
	id := anaID(t, handler, projectID)

	// act
	searched := searchPeople(t, handler, otherToken, projectID, "karo")
	added := addMember(t, handler, otherToken, projectID, `{"userId": "`+id+`"}`)

	// assert
	assert.Equal(t, http.StatusNotFound, searched.Code)
	assert.Equal(t, http.StatusNotFound, added.Code)
	assert.JSONEq(t, `[]`, projects(t, handler, http.MethodGet, otherToken, "").Body.String())
}

func joinProject(t *testing.T, handler http.Handler, token string, projectID string) *httptest.ResponseRecorder {
	t.Helper()

	return request(t, handler, http.MethodPost, "/api/projects/"+projectID+"/join", http.Header{"Authorization": {"Bearer " + token}})
}

func changeJoining(t *testing.T, handler http.Handler, token string, projectID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{"Content-Type": {"application/json"}, "Authorization": {"Bearer " + token}}

	return requestWithBody(t, handler, http.MethodPut, "/api/projects/"+projectID+"/joining", header, strings.NewReader(body))
}

func openProject(t *testing.T, handler http.Handler) string {
	t.Helper()

	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Summer fair", "visibility": "public", "joining": "open"}`)
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))

	return project.ID
}

func TestSomeoneJoinsAnOpenProject(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := openProject(t, handler)

	// act
	response := joinProject(t, handler, otherToken, projectID)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Summer fair", "visibility": "public", "joining": "open"}`, response.Body.String())

	listed := projects(t, handler, http.MethodGet, otherToken, "")
	assert.Contains(t, listed.Body.String(), projectID)
}

func TestOthersSeeThatAPublicProjectIsOpen(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := openProject(t, handler)

	// act
	response := get(t, handler, "/api/projects/public", http.Header{"Authorization": {"Bearer " + otherToken}})

	// assert
	assert.JSONEq(t, `[{"id": "`+projectID+`", "name": "Summer fair", "joining": "open", "asked": false}]`, response.Body.String())
}

func TestAProjectYouCantJoinIsNotFound(t *testing.T) {
	tests := []struct {
		name      string
		projectID func(http.Handler) string
	}{
		{name: "closed", projectID: func(handler http.Handler) string { return createdProjectWith(t, handler, "public") }},
		{name: "private", projectID: func(handler http.Handler) string { return createdProject(t, handler) }},
		{name: "no such project", projectID: func(http.Handler) string { return "01a14180-d957-7cc5-af29-7f1fe18d89bb" }},
		{name: "not an ID", projectID: func(http.Handler) string { return "garden" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := joinProject(t, handler, otherToken, test.projectID(handler))

			// assert
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String())
		})
	}
}

func TestAMemberOpensAProject(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProjectWith(t, handler, "public")

	// act
	response := changeJoining(t, handler, validToken, projectID, `{"joining": "open"}`)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id": "`+projectID+`", "name": "Garden", "visibility": "public", "joining": "open"}`, response.Body.String())
}

func TestSomeoneOutsideCantChangeHowPeopleJoin(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProjectWith(t, handler, "public")

	// act
	response := changeJoining(t, handler, otherToken, projectID, `{"joining": "open"}`)

	// assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(t, http.StatusNotFound, joinProject(t, handler, otherToken, projectID).Code)
}

func TestAJoiningFroglyCantUseIsABadRequest(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := changeJoining(t, handler, validToken, projectID, `{"joining": "open"}`)

	// assert
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.Contains(t, response.Body.String(), "only a public project can be open to join or take requests")
}

func requestProject(t *testing.T, handler http.Handler) string {
	t.Helper()

	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Summer fair", "visibility": "public", "joining": "request"}`)
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))

	return project.ID
}

func anaAsks(t *testing.T, handler http.Handler, projectID string) *httptest.ResponseRecorder {
	t.Helper()

	return request(t, handler, http.MethodPost, "/api/projects/"+projectID+"/requests", http.Header{"Authorization": {"Bearer " + otherToken}})
}

func answer(t *testing.T, handler http.Handler, method string, path string) *httptest.ResponseRecorder {
	t.Helper()

	return request(t, handler, method, path, http.Header{"Authorization": {"Bearer " + validToken}})
}

// members answer a request by the asker's ID, which only the list of
// requests gives
func anaAsked(t *testing.T, handler http.Handler, projectID string) string {
	t.Helper()

	require.Equal(t, http.StatusOK, anaAsks(t, handler, projectID).Code)

	var waiting []struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(answer(t, handler, http.MethodGet, "/api/projects/"+projectID+"/requests").Body.Bytes(), &waiting))
	require.Len(t, waiting, 1)

	return waiting[0].ID
}

func TestTheAskerSeesTheyAsked(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := requestProject(t, handler)

	// act
	response := anaAsks(t, handler, projectID)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"asked": true}`, response.Body.String())

	listed := get(t, handler, "/api/projects/public", http.Header{"Authorization": {"Bearer " + otherToken}})
	assert.JSONEq(t, `[{"id": "`+projectID+`", "name": "Summer fair", "joining": "request", "asked": true}]`, listed.Body.String())
}

func TestMembersSeeWhoAsked(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := requestProject(t, handler)
	require.Equal(t, http.StatusOK, anaAsks(t, handler, projectID).Code)

	// act
	response := answer(t, handler, http.MethodGet, "/api/projects/"+projectID+"/requests")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)

	var waiting []struct {
		ID      string    `json:"id"`
		Name    string    `json:"name"`
		Email   string    `json:"email"`
		AskedAt time.Time `json:"askedAt"`
	}

	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &waiting))
	require.Len(t, waiting, 1)
	assert.NotEmpty(t, waiting[0].ID)
	assert.Equal(t, "Ana", waiting[0].Name)
	assert.WithinDuration(t, time.Now(), waiting[0].AskedAt, time.Minute)
}

func TestAYesAnswersWithTheNewMember(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := requestProject(t, handler)
	anaID := anaAsked(t, handler, projectID)

	// act
	response := answer(t, handler, http.MethodPost, "/api/projects/"+projectID+"/requests/"+anaID+"/accept")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"id": "`+anaID+`", "name": "Ana", "email": ""}`, response.Body.String())
	assert.Contains(t, projects(t, handler, http.MethodGet, otherToken, "").Body.String(), projectID)
	assert.JSONEq(t, `[]`, answer(t, handler, http.MethodGet, "/api/projects/"+projectID+"/requests").Body.String())
}

func TestANoAnswersWithNoContent(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := requestProject(t, handler)
	anaID := anaAsked(t, handler, projectID)

	// act
	response := answer(t, handler, http.MethodDelete, "/api/projects/"+projectID+"/requests/"+anaID)

	// assert
	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())
	assert.Empty(t, response.Header().Get("Content-Type"))
	assert.JSONEq(t, `[]`, answer(t, handler, http.MethodGet, "/api/projects/"+projectID+"/requests").Body.String())
	assert.JSONEq(t, `[]`, projects(t, handler, http.MethodGet, otherToken, "").Body.String())
}

func TestAProjectThatDoesntTakeRequestsIsNotFoundForAsking(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := openProject(t, handler)

	// act
	response := anaAsks(t, handler, projectID)

	// assert
	assert.Equal(t, http.StatusNotFound, response.Code)
	assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"), "frogly's answer, not a missing route")
}

func TestAnAnswerNobodyWaitsForIsNotFound(t *testing.T) {
	for _, userID := range []string{"01a14180-d957-7cc5-af29-7f1fe18d89bb", "ana"} {
		t.Run(userID, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})
			projectID := requestProject(t, handler)

			// act
			accepted := answer(t, handler, http.MethodPost, "/api/projects/"+projectID+"/requests/"+userID+"/accept")
			declined := answer(t, handler, http.MethodDelete, "/api/projects/"+projectID+"/requests/"+userID)

			// assert
			assert.Equal(t, http.StatusNotFound, accepted.Code)
			assert.Equal(t, http.StatusNotFound, declined.Code)
			assert.Equal(t, "application/problem+json", accepted.Header().Get("Content-Type"), "frogly's answer, not a missing route")
			assert.Equal(t, "application/problem+json", declined.Header().Get("Content-Type"))
		})
	}
}

func TestRequestsAreNotFoundForSomeoneOutside(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := requestProject(t, handler)
	anaID := anaAsked(t, handler, projectID)
	header := http.Header{"Authorization": {"Bearer " + otherToken}}

	// act
	listed := get(t, handler, "/api/projects/"+projectID+"/requests", header)
	accepted := request(t, handler, http.MethodPost, "/api/projects/"+projectID+"/requests/"+anaID+"/accept", header)
	declined := request(t, handler, http.MethodDelete, "/api/projects/"+projectID+"/requests/"+anaID, header)

	// assert
	for _, response := range []*httptest.ResponseRecorder{listed, accepted, declined} {
		assert.Equal(t, http.StatusNotFound, response.Code)
		assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"), "frogly's answer, not a missing route")
	}

	waiting := answer(t, handler, http.MethodGet, "/api/projects/"+projectID+"/requests")
	assert.Contains(t, waiting.Body.String(), anaID, "the request still waits")
}

func leaveProject(t *testing.T, handler http.Handler, token string, projectID string) *httptest.ResponseRecorder {
	t.Helper()

	return request(t, handler, http.MethodPost, "/api/projects/"+projectID+"/leave", http.Header{"Authorization": {"Bearer " + token}})
}

func TestAMemberLeavesAProject(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := leaveProject(t, handler, validToken, projectID)

	// assert
	assert.Equal(t, http.StatusNoContent, response.Code)
	assert.Empty(t, response.Body.String())

	listed := projects(t, handler, http.MethodGet, validToken, "")
	assert.JSONEq(t, `[]`, listed.Body.String())
}

func TestAProjectYouCantLeaveIsNotFound(t *testing.T) {
	tests := []struct {
		name      string
		token     string
		projectID func(http.Handler) string
	}{
		{name: "not a member", token: otherToken, projectID: func(handler http.Handler) string { return createdProject(t, handler) }},
		{name: "no such project", token: validToken, projectID: func(http.Handler) string { return "01a14180-d957-7cc5-af29-7f1fe18d89bb" }},
		{name: "not an ID", token: validToken, projectID: func(http.Handler) string { return "garden" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})

			// act
			response := leaveProject(t, handler, test.token, test.projectID(handler))

			// assert
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String())
		})
	}
}

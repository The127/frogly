package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/server"
)

const (
	validToken   = "valid-token"
	outageToken  = "token-during-an-outage"
	refusedToken = "forged"
)

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, accessToken string) (authn.Person, error) {
	switch accessToken {
	case validToken:
		return authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "Karo"}, nil

	case outageToken:
		return authn.Person{}, errors.New("fetching the issuer's keys: connection refused")

	default:
		return authn.Person{}, errors.Join(authn.ErrUnauthenticated, errors.New("not the valid token"))
	}
}

func newServer(log *bytes.Buffer) http.Handler {
	logger := slog.New(slog.NewJSONHandler(log, nil))

	return server.New(
		config.OIDC{Issuer: "http://issuer.example", ClientID: "frogly"},
		fakeVerifier{},
		logger,
	)
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
	handler := newServer(&bytes.Buffer{})

	// act
	response := get(t, handler, "/api/config", nil)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "application/json", response.Header().Get("Content-Type"))
	assert.JSONEq(t, `{"issuer": "http://issuer.example", "clientId": "frogly"}`, response.Body.String())
}

func TestMeGreetsThePersonByName(t *testing.T) {
	// arrange
	handler := newServer(&bytes.Buffer{})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + validToken}})

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.Equal(t, "no-store", response.Header().Get("Cache-Control"))
	assert.JSONEq(t, `{"name": "Karo"}`, response.Body.String())
}

func TestMeAcceptsTheSchemeInAnyCase(t *testing.T) {
	// arrange
	handler := newServer(&bytes.Buffer{})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"bearer " + validToken}})

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
}

func TestMeIsUnavailableWhenTheTokenCantBeChecked(t *testing.T) {
	// arrange
	handler := newServer(&bytes.Buffer{})

	// act
	response := get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + outageToken}})

	// assert
	assert.Equal(t, http.StatusServiceUnavailable, response.Code)
	assert.Empty(t, response.Header().Get("WWW-Authenticate"))
	assert.JSONEq(t, `{"type": "about:blank", "title": "Service Unavailable", "status": 503}`, response.Body.String())
}

func TestOtherMethodsAreNotAllowed(t *testing.T) {
	for _, path := range []string{"/api/config", "/api/me"} {
		t.Run(path, func(t *testing.T) {
			// arrange
			handler := newServer(&bytes.Buffer{})

			// act
			response := request(t, handler, http.MethodPost, path, nil)

			// assert
			assert.Equal(t, http.StatusMethodNotAllowed, response.Code)
			assert.Equal(t, "GET, HEAD", response.Header().Get("Allow"))
			assert.JSONEq(t, `{"type": "about:blank", "title": "Method Not Allowed", "status": 405}`, response.Body.String())
		})
	}
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
			handler := newServer(&bytes.Buffer{})

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
	handler := newServer(&bytes.Buffer{})

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

	handler := newServer(&log)

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

	handler := newServer(&log)

	// act
	get(t, handler, "/api/me", http.Header{"Authorization": {"Bearer " + refusedToken}})

	// assert
	assert.Contains(t, log.String(), `"status":401`)
	assert.NotContains(t, log.String(), refusedToken)
}

func TestTheServerChoosesTheRequestID(t *testing.T) {
	// arrange
	handler := newServer(&bytes.Buffer{})
	sent := http.Header{"X-Request-Id": {"chosen-by-the-client"}}

	// act
	first := get(t, handler, "/api/config", sent)
	second := get(t, handler, "/api/config", sent)

	// assert
	assert.NotEqual(t, "chosen-by-the-client", first.Header().Get("X-Request-Id"))
	assert.NotEqual(t, first.Header().Get("X-Request-Id"), second.Header().Get("X-Request-Id"))
}

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

const validToken = "valid-token"

type fakeVerifier struct{}

func (fakeVerifier) Verify(_ context.Context, accessToken string) (authn.Person, error) {
	if accessToken != validToken {
		return authn.Person{}, errors.Join(authn.ErrUnauthenticated, errors.New("not the valid token"))
	}

	return authn.Person{Issuer: "http://issuer.example", Subject: "karo-id", Name: "Karo"}, nil
}

func newServer(log *bytes.Buffer) http.Handler {
	logger := slog.New(slog.NewJSONHandler(log, nil))

	return server.New(
		config.OIDC{Issuer: "http://issuer.example", ClientID: "frogly"},
		fakeVerifier{},
		logger,
	)
}

func get(t *testing.T, handler http.Handler, path string, header http.Header) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequestWithContext(t.Context(), http.MethodGet, path, nil)
	for name, values := range header {
		request.Header[name] = values
	}

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)

	return recorder
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
	assert.JSONEq(t, `{"name": "Karo"}`, response.Body.String())
}

func TestMeRefusesRequestsWithoutAValidToken(t *testing.T) {
	tests := []struct {
		name          string
		authorization string
		wantChallenge string
	}{
		{name: "no token", authorization: "", wantChallenge: `Bearer`},
		{name: "another scheme", authorization: "Basic a2Fybzpmcm9nbHk=", wantChallenge: `Bearer`},
		{name: "invalid token", authorization: "Bearer forged", wantChallenge: `Bearer error="invalid_token"`},
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

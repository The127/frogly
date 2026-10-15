package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createdProject(t *testing.T, handler http.Handler) string {
	t.Helper()

	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Garden"}`)
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))

	return project.ID
}

func ideas(t *testing.T, handler http.Handler, method string, token string, projectID string, body string) *http.Response {
	t.Helper()

	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + token},
	}

	return requestWithBody(t, handler, method, "/api/projects/"+projectID+"/ideas", header, strings.NewReader(body)).Result()
}

func body(t *testing.T, response *http.Response) string {
	t.Helper()

	var content bytes.Buffer
	_, err := content.ReadFrom(response.Body)
	require.NoError(t, err)

	return content.String()
}

func TestAPlantedIdeaIsInTheGarden(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	planted := ideas(t, handler, http.MethodPost, validToken, projectID, `{"text": "  search by any word "}`)

	// assert
	require.Equal(t, http.StatusCreated, planted.StatusCode)
	assert.Equal(t, "no-store", planted.Header.Get("Cache-Control"))

	var idea struct {
		ID        string    `json:"id"`
		Text      string    `json:"text"`
		PlantedBy string    `json:"plantedBy"`
		PlantedAt time.Time `json:"plantedAt"`
	}

	answer := body(t, planted)
	require.NoError(t, json.Unmarshal([]byte(answer), &idea))
	assert.NotEmpty(t, idea.ID)
	assert.Equal(t, "search by any word", idea.Text)
	assert.Equal(t, "Karo", idea.PlantedBy)
	assert.WithinDuration(t, time.Now(), idea.PlantedAt, time.Minute)

	listed := ideas(t, handler, http.MethodGet, validToken, projectID, "")
	assert.Equal(t, http.StatusOK, listed.StatusCode)
	assert.JSONEq(t, "["+answer+"]", body(t, listed))
}

func TestAGardenStartsEmpty(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := ideas(t, handler, http.MethodGet, validToken, projectID, "")

	// assert
	assert.Equal(t, http.StatusOK, response.StatusCode)
	assert.JSONEq(t, `[]`, body(t, response))
}

func TestAGardenYouCantSeeIsNotFound(t *testing.T) {
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
			for _, method := range []string{http.MethodGet, http.MethodPost} {
				// act
				response := ideas(t, handler, method, test.token, test.projectID, `{"text": "let me in"}`)

				// assert
				assert.Equal(t, http.StatusNotFound, response.StatusCode, method)
				assert.Equal(t, "application/problem+json", response.Header.Get("Content-Type"), method)
			}
		})
	}
}

func TestAnIdeaFroglyCantUseIsABadRequest(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := ideas(t, handler, http.MethodPost, validToken, projectID, `{"text": "  "}`)

	// assert
	assert.Equal(t, http.StatusBadRequest, response.StatusCode)
	assert.JSONEq(t, `{"type": "about:blank", "title": "Bad Request", "status": 400, "detail": "an idea needs a text"}`, body(t, response))
}

func TestPlantingNeedsAToken(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := ideas(t, handler, http.MethodPost, "", projectID, `{"text": "an idea"}`)

	// assert
	assert.Equal(t, http.StatusUnauthorized, response.StatusCode)
}

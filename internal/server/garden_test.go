package server_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createdProject(t *testing.T, handler http.Handler) string {
	t.Helper()

	return createdProjectWith(t, handler, "private")
}

func createdProjectWith(t *testing.T, handler http.Handler, visibility string) string {
	t.Helper()

	joining := ""
	if visibility == "public" {
		joining = `, "joining": "closed"`
	}

	created := projects(t, handler, http.MethodPost, validToken, `{"name": "Garden", "visibility": "`+visibility+`"`+joining+`}`)
	require.Equal(t, http.StatusCreated, created.Code)

	var project struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(created.Body.Bytes(), &project))

	return project.ID
}

func ideas(t *testing.T, handler http.Handler, method string, token string, projectID string, body string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{
		"Content-Type":  {"application/json"},
		"Authorization": {"Bearer " + token},
	}

	return requestWithBody(t, handler, method, "/api/projects/"+projectID+"/ideas", header, strings.NewReader(body))
}

func TestAPlantedIdeaIsInTheGarden(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	planted := ideas(t, handler, http.MethodPost, validToken, projectID, `{"text": "  search by any word "}`)

	// assert
	require.Equal(t, http.StatusCreated, planted.Code)
	assert.Equal(t, "no-store", planted.Header().Get("Cache-Control"))

	var idea struct {
		ID        string    `json:"id"`
		Text      string    `json:"text"`
		PlantedBy string    `json:"plantedBy"`
		PlantedAt time.Time `json:"plantedAt"`
		People    int       `json:"people"`
	}

	answer := planted.Body.String()
	require.NoError(t, json.Unmarshal([]byte(answer), &idea))
	assert.NotEmpty(t, idea.ID)
	assert.Equal(t, "search by any word", idea.Text)
	assert.Equal(t, "Karo", idea.PlantedBy)
	assert.WithinDuration(t, time.Now(), idea.PlantedAt, time.Minute)
	assert.Equal(t, 1, idea.People)

	listed := ideas(t, handler, http.MethodGet, validToken, projectID, "")
	assert.Equal(t, http.StatusOK, listed.Code)
	assert.JSONEq(t, "["+answer+"]", listed.Body.String())
}

func TestAGardenStartsEmpty(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := ideas(t, handler, http.MethodGet, validToken, projectID, "")

	// assert
	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `[]`, response.Body.String())
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
				assert.Equal(t, http.StatusNotFound, response.Code, method)
				assert.Equal(t, "application/problem+json", response.Header().Get("Content-Type"), method)
				// nothing that tells the cases apart
				assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String(), method)
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
	assert.Equal(t, http.StatusBadRequest, response.Code)
	assert.JSONEq(t, `{"type": "about:blank", "title": "Bad Request", "status": 400, "detail": "an idea needs a text"}`, response.Body.String())
}

func TestABodyThatIsntAnIdeaIsTheSameBadRequestForEveryone(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	for _, token := range []string{validToken, otherToken} {
		// act
		response := ideas(t, handler, http.MethodPost, token, projectID, `{"name": "not an idea"}`)

		// assert
		assert.Equal(t, http.StatusBadRequest, response.Code, token)
		assert.JSONEq(t, `{"type": "about:blank", "title": "Bad Request", "status": 400, "detail": "the body isn't the JSON this request takes"}`, response.Body.String(), token)
	}
}

func TestPlantingNeedsAToken(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)

	// act
	response := ideas(t, handler, http.MethodPost, "", projectID, `{"text": "an idea"}`)

	// assert
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func meToo(t *testing.T, handler http.Handler, token string, projectID string, ideaID string) *httptest.ResponseRecorder {
	t.Helper()

	header := http.Header{"Authorization": {"Bearer " + token}}

	return request(t, handler, http.MethodPost, "/api/projects/"+projectID+"/ideas/"+ideaID+"/metoo", header)
}

func plantedIdeaID(t *testing.T, handler http.Handler, projectID string) string {
	t.Helper()

	planted := ideas(t, handler, http.MethodPost, validToken, projectID, `{"text": "search by any word"}`)
	require.Equal(t, http.StatusCreated, planted.Code)

	var idea struct {
		ID string `json:"id"`
	}

	require.NoError(t, json.Unmarshal(planted.Body.Bytes(), &idea))

	return idea.ID
}

func TestAMeTooAnswersWithTheIdeaAndItsPeople(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)
	require.Equal(t, http.StatusOK, addMember(t, handler, validToken, projectID, `{"userId": "`+anaID(t, handler, projectID)+`"}`).Code)
	ideaID := plantedIdeaID(t, handler, projectID)

	// act
	response := meToo(t, handler, otherToken, projectID, ideaID)

	// assert
	assert.Equal(t, http.StatusOK, response.Code)

	var idea struct {
		ID        string `json:"id"`
		PlantedBy string `json:"plantedBy"`
		People    int    `json:"people"`
	}

	require.NoError(t, json.Unmarshal(response.Body.Bytes(), &idea))
	assert.Equal(t, ideaID, idea.ID)
	assert.Equal(t, "Karo", idea.PlantedBy)
	assert.Equal(t, 2, idea.People)

	listed := ideas(t, handler, http.MethodGet, validToken, projectID, "")
	assert.Contains(t, listed.Body.String(), `"people":2`)
}

func TestAMeTooYouCantSayIsNotFound(t *testing.T) {
	tests := []struct {
		name   string
		token  string
		ideaID func(string) string
	}{
		{name: "not a member", token: otherToken, ideaID: func(id string) string { return id }},
		{name: "no such idea", token: validToken, ideaID: func(string) string { return "01a14180-d957-7cc5-af29-7f1fe18d89bb" }},
		{name: "not an ID", token: validToken, ideaID: func(string) string { return "search" }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			handler := newServer(t, &bytes.Buffer{})
			projectID := createdProjectWith(t, handler, "public")
			ideaID := plantedIdeaID(t, handler, projectID)

			// act
			response := meToo(t, handler, test.token, projectID, test.ideaID(ideaID))

			// assert
			assert.Equal(t, http.StatusNotFound, response.Code)
			assert.JSONEq(t, `{"type": "about:blank", "title": "Not Found", "status": 404}`, response.Body.String())
		})
	}
}

func TestAMeTooNeedsAToken(t *testing.T) {
	// arrange
	handler := newServer(t, &bytes.Buffer{})
	projectID := createdProject(t, handler)
	ideaID := plantedIdeaID(t, handler, projectID)

	// act
	response := meToo(t, handler, "", projectID, ideaID)

	// assert
	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

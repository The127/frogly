//go:build integration

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

type project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func call(ctx context.Context, method string, path string, token string, body any) *http.Response {
	var content bytes.Buffer
	if body != nil {
		Expect(json.NewEncoder(&content).Encode(body)).To(Succeed())
	}

	request, err := http.NewRequestWithContext(ctx, method, froglyURL+path, &content)
	Expect(err).NotTo(HaveOccurred())

	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	Expect(err).NotTo(HaveOccurred())
	DeferCleanup(response.Body.Close)

	return response
}

func projectsOf(ctx context.Context, token string) []project {
	response := call(ctx, http.MethodGet, "/api/projects", token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var projects []project
	Expect(json.NewDecoder(response.Body).Decode(&projects)).To(Succeed())

	return projects
}

func createProject(ctx context.Context, token string, name string) project {
	response := call(ctx, http.MethodPost, "/api/projects", token, map[string]string{"name": name})
	Expect(response).To(HaveHTTPStatus(http.StatusCreated))

	var created project
	Expect(json.NewDecoder(response.Body).Decode(&created)).To(Succeed())

	return created
}

var _ = Describe("creating a project", func() {
	It("shows the project in your list and nobody else's", func(ctx SpecContext) {
		name := "Garden " + time.Now().Format(time.RFC3339Nano)

		created := createProject(ctx, accessToken, name)
		Expect(created.ID).NotTo(BeEmpty())
		Expect(created.Name).To(Equal(name))

		Expect(projectsOf(ctx, accessToken)).To(ContainElement(created))
		Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(created))
	}, SpecTimeout(10*time.Second))
})

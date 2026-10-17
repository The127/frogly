//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

type visibleProject struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

func createVisibleProject(ctx context.Context, token string, name string, visibility string) visibleProject {
	response := call(ctx, http.MethodPost, "/api/projects", token, map[string]string{"name": name, "visibility": visibility})
	Expect(response).To(HaveHTTPStatus(http.StatusCreated))

	var created visibleProject
	Expect(json.NewDecoder(response.Body).Decode(&created)).To(Succeed())

	return created
}

func changeVisibility(ctx context.Context, token string, projectID string, visibility string) *http.Response {
	return call(ctx, http.MethodPut, "/api/projects/"+projectID+"/visibility", token, map[string]string{"visibility": visibility})
}

// a map, so a test also sees a field a non-member mustn't get
func publicProjectsFor(ctx context.Context, token string) []map[string]any {
	response := call(ctx, http.MethodGet, "/api/projects/public", token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var listed []map[string]any
	Expect(json.NewDecoder(response.Body).Decode(&listed)).To(Succeed())

	return listed
}

var _ = Describe("making a project public or private", func() {
	Context("for someone outside the project", func() {
		It("lists a public project by its name only", func(ctx SpecContext) {
			fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")
			Expect(fair.Visibility).To(Equal("public"))

			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(map[string]any{"id": fair.ID, "name": "Summer fair"}))
		}, SpecTimeout(10*time.Second))

		It("doesn't list a private project", func(ctx SpecContext) {
			secret := createVisibleProject(ctx, accessToken, "Secret", "private")

			Expect(publicProjectsFor(ctx, otherAccessToken)).NotTo(ContainElement(HaveKeyWithValue("id", secret.ID)))
		}, SpecTimeout(10*time.Second))

		It("keeps the garden of a public project closed", func(ctx SpecContext) {
			fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")

			garden := call(ctx, http.MethodGet, "/api/projects/"+fair.ID+"/ideas", otherAccessToken, nil)
			Expect(garden).To(HaveHTTPStatus(http.StatusNotFound))
		}, SpecTimeout(10*time.Second))
	})

	Context("for a member", func() {
		It("doesn't list their own public project among the others", func(ctx SpecContext) {
			fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")

			Expect(publicProjectsFor(ctx, accessToken)).NotTo(ContainElement(HaveKeyWithValue("id", fair.ID)))
			Expect(projectsOf(ctx, accessToken)).To(ContainElement(project{ID: fair.ID, Name: "Summer fair"}))
		}, SpecTimeout(10*time.Second))

		It("hides a public project once they make it private, and shows it again once it's public", func(ctx SpecContext) {
			fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")

			Expect(changeVisibility(ctx, accessToken, fair.ID, "private")).To(HaveHTTPStatus(http.StatusOK))
			Expect(publicProjectsFor(ctx, otherAccessToken)).NotTo(ContainElement(HaveKeyWithValue("id", fair.ID)))
			Expect(projectsOf(ctx, accessToken)).To(ContainElement(project{ID: fair.ID, Name: "Summer fair"}))

			Expect(changeVisibility(ctx, accessToken, fair.ID, "public")).To(HaveHTTPStatus(http.StatusOK))
			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(HaveKeyWithValue("id", fair.ID)))
		}, SpecTimeout(10*time.Second))
	})

	It("can't be changed by someone outside the project", func(ctx SpecContext) {
		fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")

		Expect(changeVisibility(ctx, otherAccessToken, fair.ID, "private")).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(HaveKeyWithValue("id", fair.ID)))

		// the same change by a member works, so the 404 above isn't a missing route
		Expect(changeVisibility(ctx, accessToken, fair.ID, "private")).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))
})

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

// a map, so a test also sees a field a non-member mustn't get
func publicProjectsFor(ctx context.Context, token string) []map[string]any {
	response := call(ctx, http.MethodGet, "/api/projects/public", token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var listed []map[string]any
	Expect(json.NewDecoder(response.Body).Decode(&listed)).To(Succeed())

	return listed
}

var _ = Describe("making a project public or private", func() {
	It("shows a public project to others by name, and hides it again once it's private", func(ctx SpecContext) {
		fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")
		secret := createVisibleProject(ctx, accessToken, "Secret", "private")
		Expect(fair.Visibility).To(Equal("public"))

		Expect(publicProjectsFor(ctx, otherAccessToken)).To(And(
			ContainElement(map[string]any{"id": fair.ID, "name": "Summer fair"}),
			Not(ContainElement(HaveKeyWithValue("id", secret.ID))),
		))
		Expect(publicProjectsFor(ctx, accessToken)).NotTo(ContainElement(HaveKeyWithValue("id", fair.ID)))

		garden := call(ctx, http.MethodGet, "/api/projects/"+fair.ID+"/ideas", otherAccessToken, nil)
		Expect(garden).To(HaveHTTPStatus(http.StatusNotFound))

		changed := call(ctx, http.MethodPut, "/api/projects/"+fair.ID+"/visibility", accessToken, map[string]string{"visibility": "private"})
		Expect(changed).To(HaveHTTPStatus(http.StatusOK))

		Expect(publicProjectsFor(ctx, otherAccessToken)).NotTo(ContainElement(HaveKeyWithValue("id", fair.ID)))
	}, SpecTimeout(10*time.Second))

	It("lets only a member change it", func(ctx SpecContext) {
		fair := createVisibleProject(ctx, accessToken, "Summer fair", "public")

		changed := call(ctx, http.MethodPut, "/api/projects/"+fair.ID+"/visibility", otherAccessToken, map[string]string{"visibility": "private"})
		Expect(changed).To(HaveHTTPStatus(http.StatusNotFound))

		Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(HaveKeyWithValue("id", fair.ID)))
	}, SpecTimeout(10*time.Second))
})

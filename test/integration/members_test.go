//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

type person struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

func search(ctx context.Context, token string, projectID string, text string) []person {
	response := call(ctx, http.MethodGet, "/api/projects/"+projectID+"/people?search="+url.QueryEscape(text), token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var found []person
	Expect(json.NewDecoder(response.Body).Decode(&found)).To(Succeed())

	return found
}

func addMember(ctx context.Context, token string, projectID string, userID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/members", token, map[string]string{"userId": userID})
}

// frogly only knows someone who has logged in once, which a first request
// does
func loggedInOnce(ctx context.Context, token string) {
	Expect(call(ctx, http.MethodGet, "/api/me", token, nil)).To(HaveHTTPStatus(http.StatusOK))
}

var _ = Describe("adding someone to your project", func() {
	It("finds them by name and lets them plan in the project", func(ctx SpecContext) {
		loggedInOnce(ctx, otherAccessToken)
		garden := createProject(ctx, accessToken, "Garden", "private")

		found := search(ctx, accessToken, garden.ID, "tests-2")
		Expect(found).To(ConsistOf(HaveField("Name", "frogly-tests-2")))

		Expect(addMember(ctx, accessToken, garden.ID, found[0].ID)).To(HaveHTTPStatus(http.StatusOK))

		Expect(projectsOf(ctx, otherAccessToken)).To(ContainElement(HaveField("ID", garden.ID)))
		planted := plant(ctx, otherAccessToken, garden.ID, "an idea from the new member")
		Expect(planted.PlantedBy).To(Equal("frogly-tests-2"))
	}, SpecTimeout(10*time.Second))

	It("doesn't find someone who is a member already", func(ctx SpecContext) {
		loggedInOnce(ctx, otherAccessToken)
		garden := createProject(ctx, accessToken, "Garden", "private")

		found := search(ctx, accessToken, garden.ID, "frogly-tests")
		Expect(found).To(ContainElement(HaveField("Name", "frogly-tests-2")))
		Expect(found).NotTo(ContainElement(HaveField("Name", "frogly-tests")))
	}, SpecTimeout(10*time.Second))

	Context("from a project you aren't a member of", func() {
		It("doesn't search", func(ctx SpecContext) {
			garden := createProject(ctx, accessToken, "Garden", "public")

			searched := call(ctx, http.MethodGet, "/api/projects/"+garden.ID+"/people?search=frogly", otherAccessToken, nil)
			Expect(searched).To(HaveHTTPStatus(http.StatusNotFound))

			// the same search by a member works, so the 404 above isn't a missing route
			search(ctx, accessToken, garden.ID, "frogly")
		}, SpecTimeout(10*time.Second))

		It("doesn't add anyone", func(ctx SpecContext) {
			loggedInOnce(ctx, otherAccessToken)
			garden := createProject(ctx, accessToken, "Garden", "public")
			found := search(ctx, accessToken, garden.ID, "frogly-tests-2")
			Expect(found).To(HaveLen(1))

			Expect(addMember(ctx, otherAccessToken, garden.ID, found[0].ID)).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", garden.ID)))

			// the same addition by a member works, so the 404 above isn't a missing route
			Expect(addMember(ctx, accessToken, garden.ID, found[0].ID)).To(HaveHTTPStatus(http.StatusOK))
		}, SpecTimeout(10*time.Second))
	})
})

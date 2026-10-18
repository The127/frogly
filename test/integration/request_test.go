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

func askToJoin(ctx context.Context, token string, projectID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/requests", token, nil)
}

func requestsFor(ctx context.Context, token string, projectID string) []person {
	response := call(ctx, http.MethodGet, "/api/projects/"+projectID+"/requests", token, nil)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var asked []person
	Expect(json.NewDecoder(response.Body).Decode(&asked)).To(Succeed())

	return asked
}

func accept(ctx context.Context, token string, projectID string, userID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/requests/"+userID+"/accept", token, nil)
}

func decline(ctx context.Context, token string, projectID string, userID string) *http.Response {
	return call(ctx, http.MethodDelete, "/api/projects/"+projectID+"/requests/"+userID, token, nil)
}

// the asker as members see them, once they asked
func asker(ctx context.Context, projectID string) person {
	Expect(askToJoin(ctx, otherAccessToken, projectID)).To(HaveHTTPStatus(http.StatusOK))

	asked := requestsFor(ctx, accessToken, projectID)
	Expect(asked).To(HaveLen(1))

	return asked[0]
}

var _ = Describe("asking to join a public project", func() {
	Context("that takes requests", func() {
		It("shows the request to the asker and to its members", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("id", fair.ID),
				HaveKeyWithValue("joining", "request"),
				HaveKeyWithValue("asked", true),
			)))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(ConsistOf(HaveField("Name", "frogly-tests-2")))
		}, SpecTimeout(10*time.Second))

		It("makes the asker a member once a member says yes", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := asker(ctx, fair.ID)

			Expect(accept(ctx, accessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(projectsOf(ctx, otherAccessToken)).To(ContainElement(HaveField("ID", fair.ID)))
			Expect(plant(ctx, otherAccessToken, fair.ID, "more stalls for crafts").PlantedBy).To(Equal("frogly-tests-2"))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(BeEmpty())
		}, SpecTimeout(10*time.Second))

		It("lets the asker ask again after a member says no", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := asker(ctx, fair.ID)

			Expect(decline(ctx, accessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(BeEmpty())
			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("id", fair.ID),
				HaveKeyWithValue("asked", false),
			)))

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))

		It("keeps one request when the asker asks again", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			asker(ctx, fair.ID)

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))
	})

	It("can't ask a project that doesn't take requests", func(ctx SpecContext) {
		for _, other := range []project{
			createPublicProject(ctx, accessToken, "Summer fair", "open"),
			createPublicProject(ctx, accessToken, "Board", "closed"),
			createProject(ctx, accessToken, "Secret", "private"),
		} {
			Expect(askToJoin(ctx, otherAccessToken, other.ID)).To(HaveHTTPStatus(http.StatusNotFound), other.Name)
		}

		// one that takes requests works, so the 404s above aren't a missing route
		fair := createPublicProject(ctx, accessToken, "Book club", "request")
		Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))

	It("doesn't show or answer requests for someone outside the project", func(ctx SpecContext) {
		fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
		waiting := asker(ctx, fair.ID)

		Expect(call(ctx, http.MethodGet, "/api/projects/"+fair.ID+"/requests", otherAccessToken, nil)).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(accept(ctx, otherAccessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
	}, SpecTimeout(10*time.Second))
})

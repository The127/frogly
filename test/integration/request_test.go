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

func waitingRequest(ctx context.Context, projectID string) person {
	Expect(askToJoin(ctx, otherAccessToken, projectID)).To(HaveHTTPStatus(http.StatusOK))

	asked := requestsFor(ctx, accessToken, projectID)
	Expect(asked).To(HaveLen(1))

	return asked[0]
}

func neverAsked(ctx context.Context, projectID string) person {
	loggedInOnce(ctx, otherAccessToken)
	found := search(ctx, accessToken, projectID, "frogly-tests-2")
	Expect(found).To(HaveLen(1))

	return found[0]
}

var _ = Describe("asking to join a public project", func() {
	Context("that takes requests", func() {
		It("shows the asker that they asked", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("id", fair.ID),
				HaveKeyWithValue("joining", "request"),
				HaveKeyWithValue("asked", false),
			)))

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("id", fair.ID),
				HaveKeyWithValue("asked", true),
			)))
		}, SpecTimeout(10*time.Second))

		It("shows the request to its members", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(requestsFor(ctx, accessToken, fair.ID)).To(ConsistOf(HaveField("Name", "frogly-tests-2")))
		}, SpecTimeout(10*time.Second))

		It("makes the asker a member who can plan in it once a member says yes", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := waitingRequest(ctx, fair.ID)

			Expect(accept(ctx, accessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(plant(ctx, otherAccessToken, fair.ID, "more stalls for crafts").PlantedBy).To(Equal("frogly-tests-2"))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(BeEmpty())
		}, SpecTimeout(10*time.Second))

		It("keeps the asker out once a member says no", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := waitingRequest(ctx, fair.ID)

			Expect(decline(ctx, accessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(BeEmpty())
		}, SpecTimeout(10*time.Second))

		It("lets the asker ask again after a no", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			Expect(decline(ctx, accessToken, fair.ID, waitingRequest(ctx, fair.ID).ID)).To(HaveHTTPStatus(http.StatusNoContent))

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))

		It("keeps one request when the asker asks again", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waitingRequest(ctx, fair.ID)

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))

		It("makes no request for someone who is a member already", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")

			Expect(askToJoin(ctx, accessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(requestsFor(ctx, accessToken, fair.ID)).To(BeEmpty())
		}, SpecTimeout(10*time.Second))

		It("has no request to say yes to for someone who never asked", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")

			Expect(accept(ctx, accessToken, fair.ID, neverAsked(ctx, fair.ID).ID)).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
		}, SpecTimeout(10*time.Second))

		It("has no request to say no to for someone who never asked", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")

			Expect(decline(ctx, accessToken, fair.ID, neverAsked(ctx, fair.ID).ID)).To(HaveHTTPStatus(http.StatusNotFound))
		}, SpecTimeout(10*time.Second))

		It("can't be asked any more once a member closes it", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			Expect(changeJoining(ctx, accessToken, fair.ID, "closed")).To(HaveHTTPStatus(http.StatusOK))

			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusNotFound))
		}, SpecTimeout(10*time.Second))
	})

	DescribeTable("can't be asked of a project that doesn't take requests",
		func(ctx SpecContext, create func(context.Context) project) {
			other := create(ctx)

			Expect(askToJoin(ctx, otherAccessToken, other.ID)).To(HaveHTTPStatus(http.StatusNotFound))

			// one that takes requests works, so the 404 above isn't a missing route
			fair := createPublicProject(ctx, accessToken, "Book club", "request")
			Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))
		},
		Entry("open", func(ctx context.Context) project { return createPublicProject(ctx, accessToken, "Summer fair", "open") }, SpecTimeout(10*time.Second)),
		Entry("closed", func(ctx context.Context) project { return createPublicProject(ctx, accessToken, "Board", "closed") }, SpecTimeout(10*time.Second)),
		Entry("private", func(ctx context.Context) project { return createProject(ctx, accessToken, "Secret", "private") }, SpecTimeout(10*time.Second)),
	)

	Context("for someone outside the project", func() {
		It("doesn't show the requests", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waitingRequest(ctx, fair.ID)

			Expect(call(ctx, http.MethodGet, "/api/projects/"+fair.ID+"/requests", otherAccessToken, nil)).To(HaveHTTPStatus(http.StatusNotFound))
		}, SpecTimeout(10*time.Second))

		It("can't say yes", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := waitingRequest(ctx, fair.ID)

			Expect(accept(ctx, otherAccessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusNotFound))

			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))

		It("can't say no", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
			waiting := waitingRequest(ctx, fair.ID)

			Expect(decline(ctx, otherAccessToken, fair.ID, waiting.ID)).To(HaveHTTPStatus(http.StatusNotFound))

			Expect(requestsFor(ctx, accessToken, fair.ID)).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))
	})
})

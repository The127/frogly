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

func createPublicProject(ctx context.Context, token string, name string, joining string) project {
	body := map[string]string{"name": name, "visibility": "public", "joining": joining}
	response := call(ctx, http.MethodPost, "/api/projects", token, body)
	Expect(response).To(HaveHTTPStatus(http.StatusCreated))

	var created project
	Expect(json.NewDecoder(response.Body).Decode(&created)).To(Succeed())

	return created
}

func join(ctx context.Context, token string, projectID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/join", token, nil)
}

func changeJoining(ctx context.Context, token string, projectID string, joining string) *http.Response {
	return call(ctx, http.MethodPut, "/api/projects/"+projectID+"/joining", token, map[string]string{"joining": joining})
}

var _ = Describe("joining a public project", func() {
	Context("that is open", func() {
		It("shows it as open to someone outside it", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "open")

			Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
				HaveKeyWithValue("id", fair.ID),
				HaveKeyWithValue("joining", "open"),
			)))
		}, SpecTimeout(10*time.Second))

		It("makes you a member who can plan in it", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "open")

			Expect(join(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(projectsOf(ctx, otherAccessToken)).To(ContainElement(HaveField("ID", fair.ID)))
			Expect(plant(ctx, otherAccessToken, fair.ID, "more stalls for crafts").PlantedBy).To(Equal("frogly-tests-2"))
		}, SpecTimeout(10*time.Second))

		It("changes nothing when you join again", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "open")
			Expect(join(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			Expect(join(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

			var joined []project
			for _, listed := range projectsOf(ctx, otherAccessToken) {
				if listed.ID == fair.ID {
					joined = append(joined, listed)
				}
			}

			Expect(joined).To(HaveLen(1))
		}, SpecTimeout(10*time.Second))

		It("can't be joined any more once it's private", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "open")
			Expect(changeVisibility(ctx, accessToken, fair.ID, "private")).To(HaveHTTPStatus(http.StatusOK))

			Expect(join(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", fair.ID)))
		}, SpecTimeout(10*time.Second))

		It("can't be joined any more once a member closes it", func(ctx SpecContext) {
			fair := createPublicProject(ctx, accessToken, "Summer fair", "open")
			Expect(changeJoining(ctx, accessToken, fair.ID, "closed")).To(HaveHTTPStatus(http.StatusOK))

			Expect(join(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusNotFound))
		}, SpecTimeout(10*time.Second))
	})

	It("doesn't let you into a closed project", func(ctx SpecContext) {
		closed := createPublicProject(ctx, accessToken, "Board", "closed")

		Expect(join(ctx, otherAccessToken, closed.ID)).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", closed.ID)))

		// an open one works, so the 404 above isn't a missing route
		open := createPublicProject(ctx, accessToken, "Summer fair", "open")
		Expect(join(ctx, otherAccessToken, open.ID)).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))

	It("doesn't let you into a private project", func(ctx SpecContext) {
		secret := createProject(ctx, accessToken, "Secret", "private")

		Expect(join(ctx, otherAccessToken, secret.ID)).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", secret.ID)))

		// an open one works, so the 404 above isn't a missing route
		open := createPublicProject(ctx, accessToken, "Summer fair", "open")
		Expect(join(ctx, otherAccessToken, open.ID)).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))

	It("lets you in once a member opens it", func(ctx SpecContext) {
		board := createPublicProject(ctx, accessToken, "Board", "closed")
		Expect(join(ctx, otherAccessToken, board.ID)).To(HaveHTTPStatus(http.StatusNotFound))

		Expect(changeJoining(ctx, accessToken, board.ID, "open")).To(HaveHTTPStatus(http.StatusOK))

		Expect(join(ctx, otherAccessToken, board.ID)).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))

	It("can't be opened by someone outside the project", func(ctx SpecContext) {
		board := createPublicProject(ctx, accessToken, "Board", "closed")

		Expect(changeJoining(ctx, otherAccessToken, board.ID, "open")).To(HaveHTTPStatus(http.StatusNotFound))
		Expect(join(ctx, otherAccessToken, board.ID)).To(HaveHTTPStatus(http.StatusNotFound))

		// the same change by a member works, so the 404 above isn't a missing route
		Expect(changeJoining(ctx, accessToken, board.ID, "open")).To(HaveHTTPStatus(http.StatusOK))
	}, SpecTimeout(10*time.Second))
})

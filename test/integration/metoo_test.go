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

func meToo(ctx context.Context, token string, projectID string, ideaID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/ideas/"+ideaID+"/metoo", token, nil)
}

func meTooIdea(ctx context.Context, token string, projectID string, ideaID string) idea {
	response := meToo(ctx, token, projectID, ideaID)
	Expect(response).To(HaveHTTPStatus(http.StatusOK))

	var said idea
	Expect(json.NewDecoder(response.Body).Decode(&said)).To(Succeed())

	return said
}

// two members, since the planter alone can't show that me too counts
func sharedProject(ctx context.Context) project {
	loggedInOnce(ctx, otherAccessToken)
	shared := createProject(ctx, accessToken, "Garden", "private")
	found := search(ctx, accessToken, shared.ID, "frogly-tests-2")
	Expect(found).To(HaveLen(1))
	Expect(addMember(ctx, accessToken, shared.ID, found[0].ID)).To(HaveHTTPStatus(http.StatusOK))

	return shared
}

var _ = Describe("saying me too to an idea", func() {
	It("counts a new idea as one person's", func(ctx SpecContext) {
		garden := createProject(ctx, accessToken, "Garden", "private")

		Expect(plant(ctx, accessToken, garden.ID, "search by any word").People).To(Equal(1))
	}, SpecTimeout(10*time.Second))

	Context("as another member", func() {
		It("counts them too, for every member", func(ctx SpecContext) {
			shared := sharedProject(ctx)
			planted := plant(ctx, accessToken, shared.ID, "search by any word")

			Expect(meTooIdea(ctx, otherAccessToken, shared.ID, planted.ID).People).To(Equal(2))

			for _, token := range []string{accessToken, otherAccessToken} {
				Expect(gardenOf(ctx, token, shared.ID)).To(ConsistOf(HaveField("People", 2)))
			}
		}, SpecTimeout(10*time.Second))

		It("counts them once when they say it again", func(ctx SpecContext) {
			shared := sharedProject(ctx)
			planted := plant(ctx, accessToken, shared.ID, "search by any word")
			meTooIdea(ctx, otherAccessToken, shared.ID, planted.ID)

			Expect(meTooIdea(ctx, otherAccessToken, shared.ID, planted.ID).People).To(Equal(2))
		}, SpecTimeout(10*time.Second))

		It("keeps the garden newest first", func(ctx SpecContext) {
			shared := sharedProject(ctx)
			older := plant(ctx, accessToken, shared.ID, "search by any word")
			newer := plant(ctx, accessToken, shared.ID, "a calmer color for done tickets")

			meTooIdea(ctx, otherAccessToken, shared.ID, older.ID)

			Expect(idsOf(gardenOf(ctx, accessToken, shared.ID))).To(Equal([]string{newer.ID, older.ID}))
		}, SpecTimeout(10*time.Second))

		It("can't say it to an idea of another project", func(ctx SpecContext) {
			shared := sharedProject(ctx)
			elsewhere := createProject(ctx, accessToken, "Elsewhere", "private")
			planted := plant(ctx, accessToken, elsewhere.ID, "search by any word")

			Expect(meToo(ctx, otherAccessToken, shared.ID, planted.ID)).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(gardenOf(ctx, accessToken, elsewhere.ID)).To(ConsistOf(HaveField("People", 1)))
		}, SpecTimeout(10*time.Second))
	})

	Context("as the planter", func() {
		It("still counts them once", func(ctx SpecContext) {
			garden := createProject(ctx, accessToken, "Garden", "private")
			planted := plant(ctx, accessToken, garden.ID, "search by any word")

			Expect(meTooIdea(ctx, accessToken, garden.ID, planted.ID).People).To(Equal(1))
		}, SpecTimeout(10*time.Second))
	})

	Context("as someone outside the project", func() {
		It("can't say it", func(ctx SpecContext) {
			garden := createProject(ctx, accessToken, "Garden", "public")
			planted := plant(ctx, accessToken, garden.ID, "search by any word")

			Expect(meToo(ctx, otherAccessToken, garden.ID, planted.ID)).To(HaveHTTPStatus(http.StatusNotFound))
			Expect(gardenOf(ctx, accessToken, garden.ID)).To(ConsistOf(HaveField("People", 1)))

			// the same by a member works, so the 404 above isn't a missing route
			Expect(meToo(ctx, accessToken, garden.ID, planted.ID)).To(HaveHTTPStatus(http.StatusOK))
		}, SpecTimeout(10*time.Second))
	})
})

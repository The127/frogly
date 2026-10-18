//go:build integration

package integration_test

import (
	"context"
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

func leave(ctx context.Context, token string, projectID string) *http.Response {
	return call(ctx, http.MethodPost, "/api/projects/"+projectID+"/leave", token, nil)
}

var _ = Describe("leaving a project", func() {
	It("takes it out of your list and keeps you out of it", func(ctx SpecContext) {
		shared := sharedProject(ctx)

		Expect(leave(ctx, otherAccessToken, shared.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(projectsOf(ctx, otherAccessToken)).NotTo(ContainElement(HaveField("ID", shared.ID)))
		Expect(call(ctx, http.MethodGet, "/api/projects/"+shared.ID+"/ideas", otherAccessToken, nil)).To(HaveHTTPStatus(http.StatusNotFound))
	}, SpecTimeout(10*time.Second))

	It("keeps the project for the others with what you planted", func(ctx SpecContext) {
		shared := sharedProject(ctx)
		planted := plant(ctx, otherAccessToken, shared.ID, "an idea before leaving")

		Expect(leave(ctx, otherAccessToken, shared.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(projectsOf(ctx, accessToken)).To(ContainElement(HaveField("ID", shared.ID)))
		Expect(idsOf(gardenOf(ctx, accessToken, shared.ID))).To(ContainElement(planted.ID))
	}, SpecTimeout(10*time.Second))

	It("lets the last member leave", func(ctx SpecContext) {
		garden := createProject(ctx, accessToken, "Garden", "private")

		Expect(leave(ctx, accessToken, garden.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(projectsOf(ctx, accessToken)).NotTo(ContainElement(HaveField("ID", garden.ID)))
	}, SpecTimeout(10*time.Second))

	It("is not found for someone who isn't a member", func(ctx SpecContext) {
		garden := createProject(ctx, accessToken, "Garden", "private")

		Expect(leave(ctx, otherAccessToken, garden.ID)).To(HaveHTTPStatus(http.StatusNotFound))

		Expect(projectsOf(ctx, accessToken)).To(ContainElement(HaveField("ID", garden.ID)))
	}, SpecTimeout(10*time.Second))
})

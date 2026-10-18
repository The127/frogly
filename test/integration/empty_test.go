//go:build integration

package integration_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

var _ = Describe("a project nobody is in", func() {
	It("turns public and open, so someone can pick it up", func(ctx SpecContext) {
		garden := createProject(ctx, accessToken, "Garden", "private")

		Expect(leave(ctx, accessToken, garden.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
			HaveKeyWithValue("id", garden.ID),
			HaveKeyWithValue("joining", "open"),
		)))
		Expect(join(ctx, otherAccessToken, garden.ID)).To(HaveHTTPStatus(http.StatusOK))
		Expect(projectsOf(ctx, otherAccessToken)).To(ContainElement(HaveField("ID", garden.ID)))
	}, SpecTimeout(10*time.Second))

	It("drops the requests nobody can answer any more", func(ctx SpecContext) {
		fair := createPublicProject(ctx, accessToken, "Summer fair", "request")
		Expect(askToJoin(ctx, otherAccessToken, fair.ID)).To(HaveHTTPStatus(http.StatusOK))

		Expect(leave(ctx, accessToken, fair.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(publicProjectsFor(ctx, otherAccessToken)).To(ContainElement(SatisfyAll(
			HaveKeyWithValue("id", fair.ID),
			HaveKeyWithValue("joining", "open"),
			HaveKeyWithValue("asked", false),
		)))
	}, SpecTimeout(10*time.Second))

	It("isn't what a project with members left in it turns into", func(ctx SpecContext) {
		shared := sharedProject(ctx)

		Expect(leave(ctx, otherAccessToken, shared.ID)).To(HaveHTTPStatus(http.StatusNoContent))

		Expect(publicProjectsFor(ctx, otherAccessToken)).NotTo(ContainElement(HaveKeyWithValue("id", shared.ID)))
	}, SpecTimeout(10*time.Second))
})

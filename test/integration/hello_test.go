//go:build integration

package integration_test

import (
	"net/http"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

var _ = Describe("seeing your name", func() {
	Context("when you are logged in", func() {
		It("greets you by name", func(ctx SpecContext) {
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, froglyURL+"/api/me", nil)
			Expect(err).NotTo(HaveOccurred())

			request.Header.Set("Authorization", "Bearer "+accessToken)

			response, err := http.DefaultClient.Do(request)
			Expect(err).NotTo(HaveOccurred())

			defer func() {
				Expect(response.Body.Close()).To(Succeed())
			}()

			Expect(response).To(HaveHTTPStatus(http.StatusOK))
			Expect(response).To(HaveHTTPBody(MatchJSON(`{"name": "frogly-tests"}`)))
		})
	})
})

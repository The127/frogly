//go:build integration

package integration_test

import (
	"net/http"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
)

var _ = Describe("using frogly", func() {
	Context("without frogly's role in your login", func() {
		It("turns you away", func(ctx SpecContext) {
			response := call(ctx, http.MethodGet, "/api/me", noRoleAccessToken, nil)
			Expect(response).To(HaveHTTPStatus(http.StatusForbidden))
		}, SpecTimeout(10*time.Second))
	})
})

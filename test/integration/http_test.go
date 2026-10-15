//go:build integration

package integration_test

import (
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("the HTTP API", func() {
	It("answers the requests in test/http as described there", func(ctx SpecContext) {
		files, err := filepath.Glob(filepath.Join("..", "http", "*.hurl"))
		Expect(err).NotTo(HaveOccurred())
		Expect(files).NotTo(BeEmpty())

		arguments := append([]string{
			"--test",
			"--variable", "frogly=" + froglyURL,
			"--variable", "issuer=" + issuerURL,
			"--secret", "token=" + accessToken,
		}, files...)

		command := exec.CommandContext(ctx, "hurl", arguments...) //nolint:gosec // the arguments are the test's own files and values
		hurl, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred(), "is Hurl installed? mise installs it")
		Eventually(hurl).WithContext(ctx).Should(gexec.Exit(0))
	}, SpecTimeout(30*time.Second))
})

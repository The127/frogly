//go:build integration

// Package integration_test runs frogly as a binary against a real Keyline
// and checks what a person can do with it, one story at a time. Run it with
// `just integration`, which starts Keyline first.
package integration_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/The127/Keyline/client"
	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
	"github.com/onsi/gomega/gexec"
)

const (
	keylineURL    = "http://localhost:8082"
	issuerURL     = keylineURL + "/oidc/" + virtualServer
	virtualServer = "frogly"
	application   = "frogly"
)

var (
	froglyURL        string
	accessToken      string
	otherAccessToken string
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "frogly")
}

var _ = BeforeSuite(func() {
	accessToken = logIn("frogly-tests", "test-service-user.pem")
	otherAccessToken = logIn("frogly-tests-2", "test-service-user-2.pem")

	binary, err := gexec.Build("github.com/the127/frogly/cmd/frogly")
	Expect(err).NotTo(HaveOccurred())

	command := exec.Command(binary) //nolint:gosec // the binary gexec just built from this repository
	command.Env = append(os.Environ(),
		// port 0 lets the system pick a free port, frogly logs which
		"FROGLY_SERVER__LISTEN_ADDRESS=127.0.0.1:0",
		"FROGLY_OIDC__ISSUER="+issuerURL,
		"FROGLY_OIDC__CLIENT_ID="+application,
		"FROGLY_LOG__FORMAT=text",
	)

	frogly, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func() {
		// frogly waits for running requests before it stops
		Expect(frogly.Terminate().Wait(10 * time.Second)).To(gexec.Exit(0))
		gexec.CleanupBuildArtifacts()
	})

	listening := regexp.MustCompile(`msg=listening address=(\S+)`)

	Eventually(func() error {
		if frogly.ExitCode() != -1 {
			return StopTrying(fmt.Sprintf("frogly exited with %d", frogly.ExitCode()))
		}

		match := listening.FindSubmatch(frogly.Err.Contents())
		if match == nil {
			return errors.New("frogly doesn't listen yet")
		}

		froglyURL = "http://" + string(match[1])

		return nil
	}).WithTimeout(10 * time.Second).Should(Succeed())
})

func logIn(serviceUser string, keyFile string) string {
	key, err := os.ReadFile(filepath.Join("..", "..", "dev", "keyline", keyFile)) //nolint:gosec // the test names its own key files
	Expect(err).NotTo(HaveOccurred())

	token, err := client.NewServiceUserTokenSource(keylineURL, virtualServer, string(key), serviceUser, serviceUser, application).Token()
	Expect(err).NotTo(HaveOccurred(), "logging in at Keyline as %s failed, run the tests with `just integration`, which starts it", serviceUser)

	return token.AccessToken
}

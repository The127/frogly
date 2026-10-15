//go:build integration

// Package integration_test runs frogly as a binary against a real Keyline
// and checks what a person can do with it, one story at a time. Start
// Keyline with `just dev` first.
package integration_test

import (
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/The127/Keyline/client"
	. "github.com/onsi/ginkgo/v2" //nolint:revive // Ginkgo's DSL reads as plain English only without the package name
	. "github.com/onsi/gomega"    //nolint:revive // same for Gomega's matchers
	"github.com/onsi/gomega/gexec"
)

const (
	keylineURL    = "http://localhost:8082"
	virtualServer = "frogly"
	application   = "frogly"
	serviceUser   = "frogly-tests"
)

var (
	// froglyURL is where the frogly under test listens.
	froglyURL string
	// accessToken is the service user's token, the way a logged in person
	// presents themselves to frogly's API.
	accessToken string

	frogly *gexec.Session
)

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "frogly")
}

var _ = BeforeSuite(func() {
	key, err := os.ReadFile(filepath.Join("..", "..", "dev", "keyline", "test-service-user.pem"))
	Expect(err).NotTo(HaveOccurred())

	token, err := client.NewServiceUserTokenSource(keylineURL, virtualServer, string(key), serviceUser, serviceUser, application).Token()
	Expect(err).NotTo(HaveOccurred(), "logging in at Keyline failed, is `just dev` running?")

	accessToken = token.AccessToken

	binary, err := gexec.Build("github.com/the127/frogly/cmd/frogly")
	Expect(err).NotTo(HaveOccurred())

	address := freeAddress()
	froglyURL = "http://" + address

	command := exec.Command(binary) //nolint:gosec // the binary gexec just built from this repository
	command.Env = append(os.Environ(),
		"FROGLY_SERVER__LISTEN_ADDRESS="+address,
		"FROGLY_OIDC__ISSUER="+keylineURL+"/oidc/"+virtualServer,
		"FROGLY_OIDC__CLIENT_ID="+application,
		"FROGLY_LOG__FORMAT=text",
	)

	frogly, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	Eventually(func() error {
		connection, err := net.Dial("tcp", address)
		if err != nil {
			return err
		}

		return connection.Close()
	}).WithTimeout(10*time.Second).Should(Succeed(), "frogly doesn't listen on %s", address)
})

var _ = AfterSuite(func() {
	if frogly != nil {
		frogly.Terminate().Wait()
	}

	gexec.CleanupBuildArtifacts()
})

// freeAddress asks the system for a port nothing listens on, so the tests
// don't depend on 8080 being free.
func freeAddress() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())

	defer func() {
		Expect(listener.Close()).To(Succeed())
	}()

	return listener.Addr().String()
}

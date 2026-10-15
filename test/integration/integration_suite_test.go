//go:build integration

// Package integration_test runs frogly as a binary against a real Keyline
// and checks what a person can do with it, one story at a time. Start
// Keyline with `just dev` first.
package integration_test

import (
	"fmt"
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
	froglyURL   string
	accessToken string
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

	frogly, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	Expect(err).NotTo(HaveOccurred())

	DeferCleanup(func() {
		// frogly waits for running requests before it stops
		Expect(frogly.Terminate().Wait(10 * time.Second)).To(gexec.Exit(0))
		gexec.CleanupBuildArtifacts()
	})

	Eventually(func() error {
		if frogly.ExitCode() != -1 {
			return StopTrying(fmt.Sprintf("frogly exited with %d", frogly.ExitCode()))
		}

		connection, err := net.DialTimeout("tcp", address, time.Second)
		if err != nil {
			return err
		}

		return connection.Close()
	}).WithTimeout(10*time.Second).Should(Succeed(), "frogly doesn't listen on %s", address)
})

// the tests mustn't depend on 8080 being free
func freeAddress() string {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	Expect(err).NotTo(HaveOccurred())

	address := listener.Addr().String()
	Expect(listener.Close()).To(Succeed())

	return address
}

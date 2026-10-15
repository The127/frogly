// Package authn finds out who a request acts for, from the access token
// it carries. Every client logs in at the OIDC issuer itself, the web app,
// the CLI and agents alike, and sends its access token. frogly only checks
// that token, it keeps no sessions of its own.
package authn

import (
	"context"
	"errors"
	"fmt"

	"github.com/coreos/go-oidc/v3/oidc"
)

// ErrUnauthenticated is wrapped into every error about a token frogly
// doesn't accept.
var ErrUnauthenticated = errors.New("unauthenticated")

// Person is who a request acts for. Issuer and Subject together identify
// them, because a subject is only unique within its issuer.
type Person struct {
	Issuer  string
	Subject string
	Name    string
}

// Verifier checks access tokens from one issuer, meant for one client.
type Verifier struct {
	verifier *oidc.IDTokenVerifier
}

// NewVerifier fetches the issuer's discovery document, so the issuer has
// to be reachable. The issuer's keys are fetched and cached on the first
// token.
func NewVerifier(ctx context.Context, issuer string, clientID string) (*Verifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering the OIDC issuer %s: %w", issuer, err)
	}

	// go-oidc names it an ID token verifier, but it checks any JWT from the
	// issuer: the signature, the issuer, the audience and the expiry
	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{ClientID: clientID}),
	}, nil
}

// Verify checks the access token and says who it belongs to.
func (verifier *Verifier) Verify(ctx context.Context, accessToken string) (Person, error) {
	token, err := verifier.verifier.Verify(ctx, accessToken)
	if err != nil {
		return Person{}, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}

	var claims struct {
		Name              string `json:"name"`
		PreferredUsername string `json:"preferred_username"`
	}

	err = token.Claims(&claims)
	if err != nil {
		return Person{}, fmt.Errorf("%w: reading the claims: %w", ErrUnauthenticated, err)
	}

	name := claims.Name
	if name == "" {
		name = claims.PreferredUsername
	}

	return Person{
		Issuer:  token.Issuer,
		Subject: token.Subject,
		Name:    name,
	}, nil
}

// Package authn finds out who a request acts for, from the access token
// it carries. Every client logs in at the OIDC issuer itself, the web app,
// the CLI and agents alike, and sends its access token. frogly only checks
// that token and keeps no sessions of its own.
package authn

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/go-jose/go-jose/v4"
)

// An ID token from the same issuer has the same signature, issuer and
// audience as an access token. Only the typ header from RFC 9068 tells the
// two apart.
const accessTokenType = "at+jwt"

// Taking the algorithms from the issuer's discovery document would rely on
// a field that describes ID tokens, and might let in a shared-secret
// algorithm.
var signingAlgorithms = []jose.SignatureAlgorithm{jose.EdDSA, jose.RS256, jose.ES256}

// Keyline and frogly can run on hosts whose clocks differ a little, and a
// fresh token mustn't look expired because of it.
const clockSkew = 30 * time.Second

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

// NewVerifier fetches the issuer's discovery document with ctx, so the
// issuer has to be reachable when frogly starts. The issuer's keys are
// fetched and cached later, on the first token, independent of ctx.
func NewVerifier(ctx context.Context, issuer string, clientID string) (*Verifier, error) {
	provider, err := oidc.NewProvider(ctx, issuer)
	if err != nil {
		return nil, fmt.Errorf("discovering the OIDC issuer %s: %w", issuer, err)
	}

	algorithms := make([]string, 0, len(signingAlgorithms))
	for _, algorithm := range signingAlgorithms {
		algorithms = append(algorithms, string(algorithm))
	}

	// go-oidc names it an ID token verifier, but it checks any JWT from the
	// issuer: the signature, the issuer, the audience and the expiry. Verify
	// adds what makes it an access token.
	return &Verifier{
		verifier: provider.Verifier(&oidc.Config{
			ClientID:             clientID,
			SupportedSigningAlgs: algorithms,
			Now: func() time.Time {
				return time.Now().Add(-clockSkew)
			},
		}),
	}, nil
}

// Verify checks the access token and says who it belongs to.
func (verifier *Verifier) Verify(ctx context.Context, accessToken string) (Person, error) {
	token, err := verifier.verifier.Verify(ctx, accessToken)
	if err != nil {
		return Person{}, fmt.Errorf("%w: %w", ErrUnauthenticated, err)
	}

	err = checkType(accessToken)
	if err != nil {
		return Person{}, err
	}

	if token.Subject == "" {
		return Person{}, fmt.Errorf("%w: the token has no subject", ErrUnauthenticated)
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

// checkType runs after the signature check, so the header it reads is the
// one the issuer signed.
func checkType(accessToken string) error {
	signed, err := jose.ParseSigned(accessToken, signingAlgorithms)
	if err != nil {
		return fmt.Errorf("%w: reading the header: %w", ErrUnauthenticated, err)
	}

	typ := signed.Signatures[0].Protected.ExtraHeaders[jose.HeaderType]
	if typ != accessTokenType {
		return fmt.Errorf("%w: the token's type is %v, not %s", ErrUnauthenticated, typ, accessTokenType)
	}

	return nil
}

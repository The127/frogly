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
// doesn't accept. An error without it means frogly couldn't check the
// token, for example because the issuer is down, and the token may be fine.
var ErrUnauthenticated = errors.New("unauthenticated")

// Person is who a request acts for. Issuer and Subject together identify
// them, because a subject is only unique within its issuer.
type Person struct {
	Issuer  string
	Subject string
	Name    string
	// Email is empty when the token doesn't carry one.
	Email string
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

	var discovery struct {
		KeysURL string `json:"jwks_uri"`
	}

	err = provider.Claims(&discovery)
	if err != nil {
		return nil, fmt.Errorf("reading the discovery document of %s: %w", issuer, err)
	}

	keys := outageReportingKeySet{remote: oidc.NewRemoteKeySet(ctx, discovery.KeysURL)}

	// go-oidc names it an ID token verifier, but it checks any JWT from the
	// issuer: the signature, the issuer, the audience and the expiry. Verify
	// adds what makes it an access token.
	return &Verifier{
		verifier: oidc.NewVerifier(issuer, keys, &oidc.Config{
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
	var outage error

	token, err := verifier.verifier.Verify(context.WithValue(ctx, outageKey{}, &outage), accessToken)
	if outage != nil {
		return Person{}, fmt.Errorf("fetching the issuer's keys: %w", outage)
	}

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
		Email             string `json:"email"`
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
		Email:   claims.Email,
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

type outageKey struct{}

// go-oidc's verifier turns every failure into a flat message, so a token it
// can't check because the issuer's keys are unreachable looks like a forged
// one. This key set sits below the verifier, where the cause is still
// there, and reports an outage through the context.
type outageReportingKeySet struct {
	remote *oidc.RemoteKeySet
}

func (keys outageReportingKeySet) VerifySignature(ctx context.Context, jwt string) ([]byte, error) {
	payload, err := keys.remote.VerifySignature(ctx, jwt)

	// RemoteKeySet only wraps an error when fetching the keys failed. A bad
	// signature or a malformed token comes without a cause.
	if err != nil && errors.Unwrap(err) != nil {
		if outage, ok := ctx.Value(outageKey{}).(*error); ok {
			*outage = err
		}
	}

	return payload, err
}

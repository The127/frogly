package authn_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/the127/frogly/internal/authn"
)

const clientID = "frogly"

// issuer is a stand-in for Keyline: it serves discovery and its keys, and
// signs tokens the way Keyline does.
type issuer struct {
	url string
	key ed25519.PrivateKey
}

func newIssuer(t *testing.T) *issuer {
	t.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	issuer := &issuer{key: private}
	mux := http.NewServeMux()

	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{
			"issuer":                                issuer.url,
			"jwks_uri":                              issuer.url + "/jwks",
			"id_token_signing_alg_values_supported": []string{"EdDSA"},
		})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: public, KeyID: "test", Algorithm: string(jose.EdDSA), Use: "sig"},
		}})
	})

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	issuer.url = server.URL

	return issuer
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}

// token signs claims, filling in a valid issuer, audience and expiry unless
// claims sets them.
func (issuer *issuer) token(t *testing.T, claims map[string]any) string {
	t.Helper()

	all := map[string]any{
		"iss": issuer.url,
		"aud": []string{clientID},
		"sub": "1d0c8b4e-0000-4000-8000-000000000000",
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute).Unix(),
	}
	for name, value := range claims {
		all[name] = value
	}

	signer, err := jose.NewSigner(
		jose.SigningKey{Algorithm: jose.EdDSA, Key: issuer.key},
		(&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", "test"),
	)
	require.NoError(t, err)

	signed, err := jwt.Signed(signer).Claims(all).Serialize()
	require.NoError(t, err)

	return signed
}

func newVerifier(t *testing.T, issuer *issuer) *authn.Verifier {
	t.Helper()

	verifier, err := authn.NewVerifier(t.Context(), issuer.url, clientID)
	require.NoError(t, err)

	return verifier
}

func TestVerifyGivesThePerson(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]any
		wantName string
	}{
		{
			name:     "name claim",
			claims:   map[string]any{"name": "Karo", "preferred_username": "karo"},
			wantName: "Karo",
		},
		{
			name:     "preferred username when there is no name",
			claims:   map[string]any{"preferred_username": "karo"},
			wantName: "karo",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			issuer := newIssuer(t)
			verifier := newVerifier(t, issuer)
			token := issuer.token(t, test.claims)

			// act
			person, err := verifier.Verify(t.Context(), token)

			// assert
			require.NoError(t, err)
			assert.Equal(t, authn.Person{
				Issuer:  issuer.url,
				Subject: "1d0c8b4e-0000-4000-8000-000000000000",
				Name:    test.wantName,
			}, person)
		})
	}
}

func TestVerifyRefusesTokensFroglyDoesntTrust(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]any
	}{
		{name: "expired", claims: map[string]any{"exp": time.Now().Add(-time.Minute).Unix()}},
		{name: "for another client", claims: map[string]any{"aud": []string{"someone-else"}}},
		{name: "from another issuer", claims: map[string]any{"iss": "http://elsewhere.example"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// arrange
			issuer := newIssuer(t)
			verifier := newVerifier(t, issuer)
			token := issuer.token(t, test.claims)

			// act
			_, err := verifier.Verify(t.Context(), token)

			// assert
			require.ErrorIs(t, err, authn.ErrUnauthenticated)
		})
	}
}

func TestVerifyRefusesATokenSignedByAnotherKey(t *testing.T) {
	// arrange
	trusted := newIssuer(t)
	verifier := newVerifier(t, trusted)
	impostor := newIssuer(t)
	impostor.url = trusted.url
	token := impostor.token(t, nil)

	// act
	_, err := verifier.Verify(t.Context(), token)

	// assert
	require.ErrorIs(t, err, authn.ErrUnauthenticated)
}

func TestVerifyRefusesGarbage(t *testing.T) {
	// arrange
	verifier := newVerifier(t, newIssuer(t))

	// act
	_, err := verifier.Verify(t.Context(), "not a token")

	// assert
	require.ErrorIs(t, err, authn.ErrUnauthenticated)
}

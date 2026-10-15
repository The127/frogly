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

const (
	clientID = "frogly"
	subject  = "1d0c8b4e-0000-4000-8000-000000000000"
)

type fakeIssuer struct {
	url    string
	key    ed25519.PrivateKey
	server *httptest.Server

	// keysDown makes the keys endpoint fail, like an issuer in an outage
	keysDown bool
}

func newIssuer(t *testing.T) *fakeIssuer {
	t.Helper()

	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)

	issuer := &fakeIssuer{key: private}
	mux := http.NewServeMux()

	// no id_token_signing_alg_values_supported, frogly mustn't depend on it
	mux.HandleFunc("GET /.well-known/openid-configuration", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(t, w, map[string]any{
			"issuer":   issuer.url,
			"jwks_uri": issuer.url + "/jwks",
		})
	})
	mux.HandleFunc("GET /jwks", func(w http.ResponseWriter, _ *http.Request) {
		if issuer.keysDown {
			http.Error(w, "down for maintenance", http.StatusServiceUnavailable)

			return
		}

		writeJSON(t, w, jose.JSONWebKeySet{Keys: []jose.JSONWebKey{
			{Key: public, KeyID: "test", Algorithm: string(jose.EdDSA), Use: "sig"},
		}})
	})

	issuer.server = httptest.NewServer(mux)
	t.Cleanup(issuer.server.Close)

	issuer.url = issuer.server.URL

	return issuer
}

func writeJSON(t *testing.T, w http.ResponseWriter, body any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	assert.NoError(t, json.NewEncoder(w).Encode(body))
}

func (issuer *fakeIssuer) token(t *testing.T, claims map[string]any) string {
	t.Helper()

	return issuer.sign(t, jose.SigningKey{Algorithm: jose.EdDSA, Key: issuer.key}, "at+jwt", claims)
}

func (issuer *fakeIssuer) sign(t *testing.T, key jose.SigningKey, typ jose.ContentType, claims map[string]any) string {
	t.Helper()

	all := map[string]any{
		"iss": issuer.url,
		"aud": []string{clientID},
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(time.Minute).Unix(),
	}
	for name, value := range claims {
		all[name] = value
	}

	signer, err := jose.NewSigner(key, (&jose.SignerOptions{}).WithType(typ).WithHeader("kid", "test"))
	require.NoError(t, err)

	signed, err := jwt.Signed(signer).Claims(all).Serialize()
	require.NoError(t, err)

	return signed
}

func newVerifier(t *testing.T, issuer *fakeIssuer) *authn.Verifier {
	t.Helper()

	verifier, err := authn.NewVerifier(t.Context(), issuer.url, clientID)
	require.NoError(t, err)

	return verifier
}

func TestVerifyGivesThePerson(t *testing.T) {
	tests := []struct {
		name      string
		claims    map[string]any
		wantName  string
		wantEmail string
	}{
		{
			name:      "name claim",
			claims:    map[string]any{"name": "Karo", "preferred_username": "karo", "email": "karo@frogly.local"},
			wantName:  "Karo",
			wantEmail: "karo@frogly.local",
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
				Subject: subject,
				Name:    test.wantName,
				Email:   test.wantEmail,
			}, person)
		})
	}
}

func TestVerifyRefusesTokensFroglyDoesntTrust(t *testing.T) {
	tests := []struct {
		name   string
		claims map[string]any
	}{
		{name: "expired", claims: map[string]any{"exp": time.Now().Add(-time.Hour).Unix()}},
		{name: "without expiry", claims: map[string]any{"exp": nil}},
		{name: "not valid yet", claims: map[string]any{"nbf": time.Now().Add(time.Hour).Unix()}},
		{name: "for another client", claims: map[string]any{"aud": []string{"someone-else"}}},
		{name: "from another issuer", claims: map[string]any{"iss": "http://elsewhere.example"}},
		{name: "without subject", claims: map[string]any{"sub": ""}},
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

func TestVerifyRefusesAnIDToken(t *testing.T) {
	// arrange
	issuer := newIssuer(t)
	verifier := newVerifier(t, issuer)
	token := issuer.sign(t, jose.SigningKey{Algorithm: jose.EdDSA, Key: issuer.key}, "JWT", map[string]any{"nonce": "n-0S6_WzA2Mj"})

	// act
	_, err := verifier.Verify(t.Context(), token)

	// assert
	require.ErrorIs(t, err, authn.ErrUnauthenticated)
}

func TestVerifyRefusesASymmetricSignature(t *testing.T) {
	// arrange
	issuer := newIssuer(t)
	verifier := newVerifier(t, issuer)
	sharedSecret := jose.SigningKey{Algorithm: jose.HS256, Key: []byte("a shared secret anyone could know")}
	token := issuer.sign(t, sharedSecret, "at+jwt", nil)

	// act
	_, err := verifier.Verify(t.Context(), token)

	// assert
	require.ErrorIs(t, err, authn.ErrUnauthenticated)
}

func TestVerifyTellsAnOutageFromABadToken(t *testing.T) {
	// arrange
	issuer := newIssuer(t)
	verifier := newVerifier(t, issuer)
	token := issuer.token(t, nil)
	issuer.keysDown = true

	// act
	_, err := verifier.Verify(t.Context(), token)

	// assert
	require.Error(t, err)
	require.NotErrorIs(t, err, authn.ErrUnauthenticated)
}

func TestVerifyTellsAnUnreachableIssuerFromABadToken(t *testing.T) {
	// arrange
	issuer := newIssuer(t)
	verifier := newVerifier(t, issuer)
	token := issuer.token(t, nil)
	issuer.server.Close()

	// act
	_, err := verifier.Verify(t.Context(), token)

	// assert
	require.Error(t, err)
	require.NotErrorIs(t, err, authn.ErrUnauthenticated)
}

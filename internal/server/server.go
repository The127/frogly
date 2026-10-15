// Package server is frogly's HTTP API.
package server

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/logging"
)

// Verifier says who an access token belongs to.
type Verifier interface {
	Verify(ctx context.Context, accessToken string) (authn.Person, error)
}

// New builds the API. Clients log in at the OIDC issuer themselves and
// send their access token, which verifier checks.
func New(oidc config.OIDC, verifier Verifier, logger *slog.Logger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"issuer":   oidc.Issuer,
			"clientId": oidc.ClientID,
		})
	})
	mux.Handle("GET /api/me", authenticated(verifier, func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"name": personFrom(r.Context()).Name,
		})
	}))
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeProblem(w, http.StatusNotFound)
	})

	return logRequests(logger, mux)
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := newRequestID()
		requestLogger := logger.With("request_id", requestID)

		w.Header().Set("X-Request-Id", requestID)

		recorder := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(recorder, r.WithContext(logging.WithLogger(r.Context(), requestLogger)))

		requestLogger.InfoContext(r.Context(), "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", recorder.status,
			"duration", time.Since(started),
		)
	})
}

func newRequestID() string {
	id := make([]byte, 8)
	_, _ = rand.Read(id) // never fails, see crypto/rand.Read

	return hex.EncodeToString(id)
}

// http.ResponseWriter doesn't tell what status a handler wrote, and the log
// line needs it.
type statusRecorder struct {
	http.ResponseWriter

	status int
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.ResponseWriter.WriteHeader(status)
}

type personKey struct{}

func authenticated(verifier Verifier, next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			// RFC 6750: a request without credentials gets the challenge
			// without an error code
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeProblem(w, http.StatusUnauthorized)

			return
		}

		person, err := verifier.Verify(r.Context(), token)
		if err != nil {
			if !errors.Is(err, authn.ErrUnauthenticated) {
				logging.FromContext(r.Context()).ErrorContext(r.Context(), "checking the token failed", "error", err)
			}

			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeProblem(w, http.StatusUnauthorized)

			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), personKey{}, person)))
	})
}

// Only handlers behind authenticated call it, so the context always
// carries a person.
func personFrom(ctx context.Context) authn.Person {
	person, _ := ctx.Value(personKey{}).(authn.Person)

	return person
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body) // the client hung up, nobody is left to tell
}

// The problem says no more than the status, so a refused token reveals
// nothing about why it was refused.
func writeProblem(w http.ResponseWriter, status int) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{ // the client hung up, nobody is left to tell
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
	})
}

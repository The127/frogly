// Package server is frogly's HTTP API.
package server

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
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

// New builds the API and serves the built frontend for every other path.
// Clients log in at the OIDC issuer themselves and send their access
// token, which verifier checks.
func New(oidc config.OIDC, verifier Verifier, logger *slog.Logger, frontend fs.FS) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"issuer":   oidc.Issuer,
			"clientId": oidc.ClientID,
		})
	})
	mux.Handle("GET /api/me", authenticated(verifier, func(w http.ResponseWriter, _ *http.Request, person authn.Person) {
		writeJSON(w, http.StatusOK, map[string]string{
			"name": person.Name,
		})
	}))
	mux.Handle("GET /", serveFrontend(frontend))

	return logRequests(logger, mux)
}

func serveFrontend(frontend fs.FS) http.Handler {
	files := http.FileServerFS(frontend)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := path.Clean(r.URL.Path)
		if strings.HasPrefix(name, "/api/") {
			writeProblem(w, http.StatusNotFound)

			return
		}

		if strings.HasPrefix(name, "/assets/") {
			// a missing asset stays a 404, because an old page asks for it and
			// index.html wouldn't help. A directory isn't an asset, and its
			// listing would be cached forever like one.
			info, err := fs.Stat(frontend, strings.TrimPrefix(name, "/"))
			if err != nil || info.IsDir() {
				http.NotFound(w, r)

				return
			}

			// Vite puts a hash of the content into every asset's name, so an
			// asset never changes
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			files.ServeHTTP(w, r)

			return
		}

		// every other path is a page the frontend shows itself, like the
		// login callback. no-cache makes the browser ask whether a new
		// build came out.
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFileFS(w, r, frontend, "index.html")
	})
}

func logRequests(logger *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		requestID := rand.Text()
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

// http.ResponseWriter doesn't tell what status a handler wrote, and the log
// line needs it.
type statusRecorder struct {
	http.ResponseWriter

	status      int
	wroteHeader bool
}

func (recorder *statusRecorder) WriteHeader(status int) {
	// net/http keeps the first status and ignores later ones
	if !recorder.wroteHeader {
		recorder.status = status
		recorder.wroteHeader = true
	}

	recorder.ResponseWriter.WriteHeader(status)
}

// Unwrap lets http.ResponseController reach the writer below, for flushing
// and deadlines.
func (recorder *statusRecorder) Unwrap() http.ResponseWriter {
	return recorder.ResponseWriter
}

func authenticated(verifier Verifier, next func(http.ResponseWriter, *http.Request, authn.Person)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// what a token unlocks is personal, no cache may keep it
		w.Header().Set("Cache-Control", "no-store")

		scheme, token, found := strings.Cut(r.Header.Get("Authorization"), " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" {
			// RFC 6750: a request without credentials gets the challenge
			// without an error code
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeProblem(w, http.StatusUnauthorized)

			return
		}

		person, err := verifier.Verify(r.Context(), token)
		if err != nil && !errors.Is(err, authn.ErrUnauthenticated) {
			// the token may be fine, a 401 would make the frontend log the
			// person out just because the issuer can't be reached
			logging.FromContext(r.Context()).ErrorContext(r.Context(), "checking the token failed", "error", err)
			writeProblem(w, http.StatusServiceUnavailable)

			return
		}

		if err != nil {
			w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
			writeProblem(w, http.StatusUnauthorized)

			return
		}

		next(w, r, person)
	})
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	write(w, "application/json", status, body)
}

// The problem says no more than the status, so a refused token reveals
// nothing about why it was refused.
func writeProblem(w http.ResponseWriter, status int) {
	write(w, "application/problem+json", status, map[string]any{
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
	})
}

func write(w http.ResponseWriter, contentType string, status int, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// an error here means the client hung up, nobody is left to tell
	_ = json.NewEncoder(w).Encode(body)
}

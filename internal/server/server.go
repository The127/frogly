// Package server is frogly's HTTP API.
package server

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"path"
	"slices"
	"strings"
	"time"

	"github.com/The127/ioc"

	"github.com/the127/frogly/internal/authn"
	"github.com/the127/frogly/internal/config"
	"github.com/the127/frogly/internal/db"
	"github.com/the127/frogly/internal/logging"
	"github.com/the127/frogly/internal/people"
	"github.com/the127/frogly/internal/projects"
	"github.com/the127/frogly/internal/repositories"
)

// Verifier says who an access token belongs to.
type Verifier interface {
	Verify(ctx context.Context, accessToken string) (authn.Person, error)
}

// New builds the API and serves the built frontend for every other path.
// Clients log in at the OIDC issuer themselves and send their access
// token, which verifier checks. services hands every authenticated request
// a db.Context of its own, from a scope that lives as long as the request.
func New(oidc config.OIDC, verifier Verifier, logger *slog.Logger, frontend fs.FS, services *ioc.DependencyProvider) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{
			"issuer":   oidc.Issuer,
			"clientId": oidc.ClientID,
		})
	})
	mux.Handle("GET /api/me", authenticated(verifier, oidc.Roles.User, services, func(_ *http.Request, caller caller) (int, any, error) {
		return http.StatusOK, map[string]string{"name": caller.user.Name()}, nil
	}))
	mux.Handle("GET /api/projects", authenticated(verifier, oidc.Roles.User, services, listProjects))
	mux.Handle("POST /api/projects", authenticated(verifier, oidc.Roles.User, services, createProject))
	mux.Handle("GET /api/projects/{id}", authenticated(verifier, oidc.Roles.User, services, getProject))
	mux.Handle("GET /api/projects/{id}/ideas", authenticated(verifier, oidc.Roles.User, services, listGarden))
	mux.Handle("POST /api/projects/{id}/ideas", authenticated(verifier, oidc.Roles.User, services, plantIdea))
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

// caller is who an authenticated request comes from, with the unit of work
// the request does its work in.
type caller struct {
	user *repositories.User
	unit db.Context
}

// endpoint does what a request asks for. It answers with a status and a
// body, or with the error that stopped it, and answerError turns that error
// into the answer.
type endpoint func(r *http.Request, caller caller) (int, any, error)

// no request frogly takes holds more than a few short fields, so anything
// bigger than this is a mistake
const maxBody = 4 << 10

// authenticated lets a request through only for a person with userRole,
// frogly's role in the login. It checks before it looks the person up, so
// frogly stores nobody who may not use it.
func authenticated(verifier Verifier, userRole string, services *ioc.DependencyProvider, next endpoint) http.Handler {
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

		if !slices.Contains(person.Roles, userRole) {
			writeProblem(w, http.StatusForbidden)

			return
		}

		scope := services.NewScope()
		defer func() {
			err := scope.Close()
			if err != nil {
				logging.FromContext(r.Context()).ErrorContext(r.Context(), "closing the request's scope failed", "error", err)
			}
		}()

		// finding the user saves on its own, so it gets its own unit of work,
		// and the handler's stays empty for what the handler records
		user, err := people.Resolve(r.Context(), ioc.GetDependency[db.Database](scope).NewContext(), person)
		if err != nil {
			fail(w, r, "finding the user failed", err)

			return
		}

		r.Body = http.MaxBytesReader(w, r.Body, maxBody)

		status, body, err := next(r, caller{user: user, unit: ioc.GetDependency[db.Context](scope)})
		if err != nil {
			answerError(w, r, err)

			return
		}

		writeJSON(w, status, body)
	})
}

// errBadBody says the request's body isn't what the endpoint takes. Its
// text is what the client reads.
var errBadBody = errors.New("the body isn't the JSON this request takes")

// decode refuses fields target doesn't have, since a client that sends
// them expects frogly to do something it doesn't. A body that can't be read
// isn't the client's mistake, so that error stays as it is.
func decode(r *http.Request, target any) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading the body: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()

	err = decoder.Decode(target)
	if err != nil {
		return errBadBody
	}

	// the decoder stops after the first value, and anything after it means
	// the body was meant as something else
	err = decoder.Decode(&struct{}{})
	if !errors.Is(err, io.EOF) {
		return errBadBody
	}

	return nil
}

// answerError is the one place that decides what a client learns about an
// error. Anything frogly doesn't expect is logged and stays a plain 500,
// since its text may say more than the client should know.
func answerError(w http.ResponseWriter, r *http.Request, err error) {
	if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
		writeProblem(w, http.StatusRequestEntityTooLarge)

		return
	}

	if errors.Is(err, errBadBody) {
		writeProblemDetail(w, http.StatusBadRequest, errBadBody.Error())

		return
	}

	if errors.Is(err, projects.ErrNotFound) {
		writeProblem(w, http.StatusNotFound)

		return
	}

	if invalid, ok := errors.AsType[*repositories.InvalidError](err); ok {
		writeProblemDetail(w, http.StatusBadRequest, invalid.Reason())

		return
	}

	fail(w, r, "the request failed", err)
}

func fail(w http.ResponseWriter, r *http.Request, what string, err error) {
	logging.FromContext(r.Context()).ErrorContext(r.Context(), what, "error", err)
	writeProblem(w, http.StatusInternalServerError)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	write(w, "application/json", status, body)
}

// The problem says no more than the status, so a refused token reveals
// nothing about why it was refused.
func writeProblem(w http.ResponseWriter, status int) {
	writeProblemDetail(w, status, "")
}

// writeProblemDetail says what is wrong with a request, for a problem the
// client can fix.
func writeProblemDetail(w http.ResponseWriter, status int, detail string) {
	problem := map[string]any{
		"type":   "about:blank",
		"title":  http.StatusText(status),
		"status": status,
	}
	if detail != "" {
		problem["detail"] = detail
	}

	write(w, "application/problem+json", status, problem)
}

func write(w http.ResponseWriter, contentType string, status int, body any) {
	w.Header().Set("Content-Type", contentType)
	w.WriteHeader(status)
	// an error here means the client hung up, nobody is left to tell
	_ = json.NewEncoder(w).Encode(body)
}

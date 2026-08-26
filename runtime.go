package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
)

var requestSequence uint64

type requestIDCtxKey struct{}

func requestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDCtxKey{}).(string); ok {
		return v
	}
	return ""
}

func serveAddress(address string, handler http.Handler) error {
	return serveHTTP(newEnterpriseServer(address, handler))
}

func serveHTTP(server *http.Server) error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- server.ListenAndServe()
	}()

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(signals)

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-signals:
		shutdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return server.Shutdown(shutdownContext)
	}
}

func newEnterpriseServer(address string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              address,
		Handler:           opsEnterpriseMiddleware(requestIDMiddleware(recoveryMiddleware(handler))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
}

// requestIDMiddleware ensures every request carries a stable correlation id.
// Callers may supply one via the X-Request-ID header; otherwise a unique,
// atomically-incremented id is generated so concurrent and back-to-back
// requests never share an id. The resolved id is placed on the request
// context so downstream handlers (including recovery) can reference it.
func requestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
		if requestID == "" {
			requestID = "req-" + strconv.FormatUint(atomic.AddUint64(&requestSequence, 1), 10)
		}
		w.Header().Set("X-Request-ID", requestID)
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), requestIDCtxKey{}, requestID)))
	})
}

// recoveryMiddleware traps panics raised by downstream handlers so a single
// faulty request cannot crash the connection or the process. It logs the
// panic with the request id (and stack trace) and returns a 500 response.
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				rid := requestIDFromContext(r.Context())
				log.Printf("panic: %v request=%s %s %s\n%s", rec, rid, r.Method, r.URL.RequestURI(), debugStack())
				if rid != "" && w.Header().Get("X-Request-ID") == "" {
					w.Header().Set("X-Request-ID", rid)
				}
				writeRecoveryResponse(w)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeRecoveryResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	// Guard against an already-committed (WriteHeader called) response so we
	// never trigger "superfluous response.WriteHeader" from inside recovery.
	w.WriteHeader(http.StatusInternalServerError)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": "internal server error"})
}

func debugStack() []byte {
	buf := make([]byte, 4096)
	n := runtime.Stack(buf, false)
	return buf[:n]
}

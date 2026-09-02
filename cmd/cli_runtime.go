package cmd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/jaslrobinson/golazo/internal/api"
	"github.com/jaslrobinson/golazo/internal/espncfb"
)

// espnLocation is US Eastern time, the timezone boundary ESPN's `dates`
// scoreboard parameter almost certainly uses (all times in captured
// responses were ET-based kickoffs). Falls back to UTC if the local tzdata
// database is unavailable, which would otherwise misclassify games near
// midnight ET. Duplicated from internal/app/commands.go since cmd cannot
// import internal/app (it would create a backwards dependency).
var espnLocation = func() *time.Location {
	loc, err := time.LoadLocation("America/New_York")
	if err != nil {
		return time.UTC
	}
	return loc
}()

// espnNow returns the current time in ESPN's date boundary (US Eastern).
// Using local or UTC time here instead would bucket late-night games onto
// the wrong calendar date relative to what the `dates` query param expects.
func espnNow() time.Time {
	return time.Now().In(espnLocation)
}

// Env vars recognized by the CLI subcommands.
const (
	EnvAgent   = "GOLAZO_AGENT"   // "1" forces compact JSON, stderr debug logging
	EnvOffline = "GOLAZO_OFFLINE" // "1" refuses any network call
)

// runtimeOpts captures the per-invocation runtime configuration.
type runtimeOpts struct {
	mock    bool
	debug   bool
	timeout time.Duration
}

// ErrOffline is returned when GOLAZO_OFFLINE is set and the subcommand needs
// network access (mock-mode callers may ignore this).
var ErrOffline = errors.New("network access disabled via GOLAZO_OFFLINE")

// agentMode returns true when GOLAZO_AGENT=1.
func agentMode() bool {
	return os.Getenv(EnvAgent) == "1"
}

// offlineMode returns true when GOLAZO_OFFLINE=1.
func offlineMode() bool {
	return os.Getenv(EnvOffline) == "1"
}

// newStderrLogger returns a slog.Logger writing to stderr at Debug level when
// debug or agent mode is on. Otherwise a no-op logger.
//
// Stdout is reserved for the JSON envelope; logs MUST go to stderr.
func newStderrLogger(debug bool) *slog.Logger {
	if !debug && !agentMode() {
		return slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelDebug})
	return slog.New(handler).With("source", "golazo")
}

// newHeadlessClient builds an espncfb.Client without the TUI's background
// version-check goroutine. Honors GOLAZO_OFFLINE by returning ErrOffline
// when the caller is not in mock mode.
//
// Returns the client as the generic api.Client interface, a context bounded
// by opts.timeout (default 15s), and the cancel function the caller MUST
// invoke.
func newHeadlessClient(opts runtimeOpts) (api.Client, context.Context, context.CancelFunc, error) {
	if offlineMode() && !opts.mock {
		// Provide a no-op cancel so callers can defer unconditionally.
		return nil, nil, func() {}, ErrOffline
	}

	timeout := opts.timeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	// In mock mode we still return a client (callers branch on opts.mock and
	// use data.Mock* sources), but we skip wiring an HTTP-bound logger when
	// not needed.
	client := espncfb.NewClient()
	client.SetLogger(newStderrLogger(opts.debug))

	return client, ctx, cancel, nil
}

// isTimeout reports whether ctx's deadline was exceeded.
func isTimeout(ctx context.Context) bool {
	return ctx != nil && errors.Is(ctx.Err(), context.DeadlineExceeded)
}

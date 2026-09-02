package espncfb

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_SetLogger_LogsRequestURL(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

	c := NewClient()
	c.SetLogger(logger)

	var out map[string]any
	if err := c.get(context.Background(), server.URL, &out); err != nil {
		t.Fatalf("get: %v", err)
	}

	if !strings.Contains(buf.String(), server.URL) {
		t.Errorf("expected debug log to contain request URL %q, got: %s", server.URL, buf.String())
	}
}

func TestClient_SetLogger_NilLoggerDoesNotPanic(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := NewClient()

	var out map[string]any
	if err := c.get(context.Background(), server.URL, &out); err != nil {
		t.Fatalf("get: %v", err)
	}
}

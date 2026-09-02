package cmd

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"
)

func TestResolveLeagues_ListsFBSConferences(t *testing.T) {
	got := resolveLeagues()
	if len(got) == 0 {
		t.Fatalf("expected at least one conference")
	}
	for i := 1; i < len(got); i++ {
		if got[i-1].ID > got[i].ID {
			t.Errorf("unsorted at %d: %d > %d", i, got[i-1].ID, got[i].ID)
		}
	}
	for _, l := range got {
		if l.Name == "" {
			t.Errorf("conference %d has empty name", l.ID)
		}
	}
}

func TestRunLeagues_EmitsEnvelope(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runLeagues(&stdout, &stderr, cliFlags{timeout: time.Second})
	if code != ExitOK {
		t.Fatalf("exit = %d, stderr=%s", code, stderr.String())
	}
	var env struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &env); err != nil {
		t.Fatalf("unmarshal: %v\nraw: %s", err, stdout.String())
	}
	if env.Status != "ok" {
		t.Errorf("status = %q", env.Status)
	}
	if env.Count != len(resolveLeagues()) {
		t.Errorf("count = %d, want %d", env.Count, len(resolveLeagues()))
	}
}

func TestRunLeagues_WorksUnderOffline(t *testing.T) {
	t.Setenv(EnvOffline, "1")

	var stdout, stderr bytes.Buffer
	code := runLeagues(&stdout, &stderr, cliFlags{timeout: time.Second})
	if code != ExitOK {
		t.Errorf("exit = %d, want %d (leagues makes no network call, must not be gated by GOLAZO_OFFLINE)", code, ExitOK)
	}
	if stdout.Len() == 0 {
		t.Errorf("stdout empty under offline")
	}
}

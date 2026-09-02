package cmd

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

// CapabilitiesSchemaVersion identifies the contract version of the capabilities
// payload. Bump when fields are added/changed so agents can pin against it.
const CapabilitiesSchemaVersion = "1"

// capabilityFlag describes a single flag in machine-readable form.
type capabilityFlag struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Default     any    `json:"default,omitempty"`
	Description string `json:"description"`
}

// capabilityCommand describes a single subcommand.
type capabilityCommand struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Args        string           `json:"args,omitempty"`
	Flags       []capabilityFlag `json:"flags"`
	Example     string           `json:"example"`
	ExitCodes   []int            `json:"exit_codes"`
}

// capabilities is the machine-readable contract published by `golazo capabilities`.
type capabilities struct {
	SchemaVersion string              `json:"schema_version"`
	Tool          string              `json:"tool"`
	Description   string              `json:"description"`
	Docs          string              `json:"docs"`
	Commands      []capabilityCommand `json:"commands"`
	ErrorCodes    map[string]int      `json:"error_codes"`
	ExitCodes     map[string]string   `json:"exit_codes"`
	EnvVars       map[string]string   `json:"env_vars"`
	Envelope      map[string]any      `json:"envelope"`
}

// buildCapabilities returns the static capabilities payload. Kept in code (not
// JSON file) so the contract stays in lockstep with the actual subcommands.
func buildCapabilities() capabilities {
	commonFlags := []capabilityFlag{
		{Name: "mock", Type: "bool", Default: false, Description: "Use bundled mock data, no network"},
		{Name: "debug", Type: "bool", Default: false, Description: "Emit debug logs to stderr"},
		{Name: "timeout", Type: "duration", Default: "15s", Description: "Overall request timeout"},
		{Name: "pretty", Type: "bool", Default: false, Description: "Indent JSON output"},
	}
	// Read-only subcommands (no network I/O) expose only --pretty so agents
	// don't see inert flags in the contract.
	prettyOnly := []capabilityFlag{
		{Name: "pretty", Type: "bool", Default: false, Description: "Indent JSON output"},
	}
	finishedFlagDefs := append([]capabilityFlag{}, commonFlags...)
	finishedFlagDefs = append(finishedFlagDefs,
		capabilityFlag{Name: "days", Type: "int", Default: DefaultFinishedDays, Description: "Number of days to look back (1..8). Default reaches the prior weekend since college football clusters Thu-Sat."},
		capabilityFlag{Name: "include-upcoming", Type: "bool", Default: false, Description: "Also include today's not-yet-started matches"},
	)

	return capabilities{
		SchemaVersion: CapabilitiesSchemaVersion,
		Tool:          "golazo",
		Description:   "JSON CLI for NCAA college football match data (live, finished, details, conferences), backed by ESPN",
		Docs:          "https://github.com/jaslrobinson/golazo/blob/main/docs/CLI.md",
		Commands: []capabilityCommand{
			{
				Name:        "live",
				Description: "List today's in-progress college football games across all FBS conferences. Often empty outside Thu-Sat — that's correct, not a failure.",
				Flags:       commonFlags,
				Example:     "golazo live",
				ExitCodes:   []int{ExitOK, ExitUpstream, ExitTimeout, ExitOffline},
			},
			{
				Name:        "finished",
				Description: "List finished college football games over a day window; optionally include today's upcoming matches",
				Flags:       finishedFlagDefs,
				Example:     "golazo finished --days 8 --include-upcoming",
				ExitCodes:   []int{ExitOK, ExitUpstream, ExitInvalidArgs, ExitTimeout, ExitOffline},
			},
			{
				Name:        "match",
				Description: "Get full game details (scoring plays, situation, player leaders, win-probability momentum, box score) by ESPN event ID.",
				Args:        "<id>",
				Flags:       commonFlags,
				Example:     "golazo match 2001 --mock",
				ExitCodes:   []int{ExitOK, ExitUpstream, ExitInvalidArgs, ExitNotFound, ExitTimeout, ExitOffline},
			},
			{
				Name:        "leagues",
				Description: "List every hardcoded FBS conference (ESPN \"group\"). No network calls.",
				Flags:       prettyOnly,
				Example:     "golazo leagues",
				ExitCodes:   []int{ExitOK},
			},
			{
				Name:        "capabilities",
				Description: "Print this machine-readable contract describing every subcommand, flag, error and exit code",
				Flags:       prettyOnly,
				Example:     "golazo capabilities | jq '.data[0].commands'",
				ExitCodes:   []int{ExitOK},
			},
		},
		ErrorCodes: map[string]int{
			string(ErrCodeInvalidArgs):   ExitInvalidArgs,
			string(ErrCodeNotFound):      ExitNotFound,
			string(ErrCodeUpstreamError): ExitUpstream,
			string(ErrCodeTimeout):       ExitTimeout,
			string(ErrCodeOffline):       ExitOffline,
		},
		ExitCodes: map[string]string{
			"0": "ok",
			"1": "upstream_error",
			"2": "invalid_args",
			"3": "not_found",
			"4": "timeout",
			"5": "offline",
		},
		EnvVars: map[string]string{
			EnvAgent:   "Forces compact JSON, enables stderr debug logging",
			EnvOffline: "Refuses any network call; subcommands return offline unless --mock is set",
		},
		Envelope: map[string]any{
			"success": map[string]any{"status": "ok", "count": "int", "data": "[]object", "degraded": "bool (optional)", "failed_dates": "[]string (optional)"},
			"error":   map[string]any{"status": "error", "code": "string", "message": "string"},
			"notes": []string{
				"Errors always go to stderr; stdout stays empty on error.",
				"Single-item responses (match <id>) still use a data array with count: 1.",
				"List output is sorted by match_time then id for deterministic ordering.",
			},
		},
	}
}

var capabilitiesFlagSet cliFlags

// runCapabilities is the testable core of the `capabilities` subcommand.
func runCapabilities(stdout, stderr io.Writer, flags cliFlags) int {
	applyPretty(flags)
	if err := WriteJSON(stdout, []capabilities{buildCapabilities()}); err != nil {
		return WriteError(stderr, ErrCodeUpstreamError, err)
	}
	return ExitOK
}

var capabilitiesCmd = &cobra.Command{
	Use:           "capabilities",
	Short:         "Print a machine-readable description of every subcommand",
	Long:          "Emits a JSON envelope describing every subcommand, flag, error code, exit code and env var. Designed for agentic tools (Claude Code, Codex, MCP servers) to self-discover the CLI contract at session start.",
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		code := runCapabilities(os.Stdout, os.Stderr, capabilitiesFlagSet)
		if code != ExitOK {
			os.Exit(code)
		}
	},
}

func init() {
	addPrettyOnlyFlag(capabilitiesCmd, &capabilitiesFlagSet)
	rootCmd.AddCommand(capabilitiesCmd)
}

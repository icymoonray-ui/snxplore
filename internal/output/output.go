// Package output renders command results for both humans and agents.
//
// The machine contract: data goes to stdout, errors go to stderr, the format
// is selected with --output (json|table), and exit codes are stable (see the
// Exit* constants). JSON mode is the agent-facing surface; table mode is a
// human convenience that falls back to JSON when a value has no table view.
package output

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// Format is the output encoding selected by --output.
type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
)

// Stable process exit codes — part of the machine-readable contract.
const (
	ExitOK       = 0
	ExitError    = 1 // generic failure
	ExitUsage    = 2 // bad invocation / flags
	ExitAuth     = 3 // authentication / authorization failure
	ExitNotFound = 4 // requested resource does not exist
	ExitAPI      = 5 // ServiceNow API returned an error
)

// Table is a simple tabular view for human (table) output.
type Table struct {
	Headers []string
	Rows    [][]string
}

// Tabular is implemented by values that can render themselves as a Table.
// In table mode, values implementing Tabular render as columns; everything
// else falls back to indented JSON.
type Tabular interface {
	TableView() Table
}

// Error is a coded error carrying a stable machine code and an exit code.
type Error struct {
	Code    string // stable identifier, e.g. "auth_failed"
	Message string
	Exit    int
}

func (e *Error) Error() string { return e.Message }

// Errorf builds a coded Error.
func Errorf(code string, exit int, format string, a ...any) *Error {
	return &Error{Code: code, Exit: exit, Message: fmt.Sprintf(format, a...)}
}

// Renderer writes results to Out and errors to Err in the chosen Format.
type Renderer struct {
	Format Format
	Out    io.Writer
	Err    io.Writer
}

// New returns a Renderer, defaulting to table format for unknown values.
func New(format Format, out, err io.Writer) *Renderer {
	if format != FormatJSON && format != FormatTable {
		format = FormatTable
	}
	return &Renderer{Format: format, Out: out, Err: err}
}

// Emit renders a successful result to stdout.
func (r *Renderer) Emit(v any) error {
	if r.Format == FormatJSON {
		return writeJSON(r.Out, v)
	}
	if t, ok := v.(Tabular); ok {
		return writeTable(r.Out, t.TableView())
	}
	return writeJSON(r.Out, v)
}

// EmitError writes a machine-readable error to stderr and returns the exit code.
func (r *Renderer) EmitError(err error) int {
	if err == nil {
		return ExitOK
	}
	exit := ExitError
	var ce *Error
	if errors.As(err, &ce) && ce.Exit != 0 {
		exit = ce.Exit
	}
	if r.Format == FormatJSON {
		_ = writeJSON(r.Err, errorEnvelope(err, ce))
	} else {
		fmt.Fprintln(r.Err, "Error:", err.Error())
	}
	return exit
}

// Note writes an advisory message to stderr in table mode. In JSON mode it is
// a no-op (the structured payload should already carry the note as a field).
func (r *Renderer) Note(msg string) {
	if msg == "" || r.Format == FormatJSON {
		return
	}
	fmt.Fprintln(r.Err, "note:", msg)
}

func errorEnvelope(err error, ce *Error) map[string]any {
	inner := map[string]any{"message": err.Error()}
	if ce != nil && ce.Code != "" {
		inner["code"] = ce.Code
	}
	return map[string]any{"error": inner}
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func writeTable(w io.Writer, t Table) error {
	tw := tabwriter.NewWriter(w, 0, 2, 2, ' ', 0)
	if len(t.Headers) > 0 {
		fmt.Fprintln(tw, strings.Join(t.Headers, "\t"))
	}
	for _, row := range t.Rows {
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

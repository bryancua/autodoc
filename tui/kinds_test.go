package tui

import (
	"testing"

	"github.com/yongjohnlee80/autodoc/core/kind"
)

// The editor's syntax for each kind, where no source language claims the file: Markdown for a note
// and a derived document's text, YAML for YAML, plain text for the rest.
func TestTheEditorsSyntaxForEachKind(t *testing.T) {
	for k, want := range map[kind.Kind]string{kind.Markdown: "Markdown", kind.Derived: "Markdown", kind.YAML: "YAML",
		kind.Text: "Plain text", kind.Registered: "Plain text"} {
		if got := syntaxOf.Of(k); got != want {
			t.Errorf("%s: %q, want %q", k, got, want)
		}
	}
}

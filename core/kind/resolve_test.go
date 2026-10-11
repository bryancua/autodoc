package kind

import (
	"strings"
	"testing"

	"github.com/yongjohnlee80/golib/parse/markdown"
)

// A resolver reads the workspace's text extensions on every call, and a format a build's deriver
// adds is derived like the built-in ones.
func TestAResolverReadsThroughItsRegistrations(t *testing.T) {
	text := []string{".log"}
	r := Resolver{Registrations: Registrations{Chunked: []string{".go"}, Derived: []string{".epub"}}, Text: func() []string { return text }}
	for p, want := range map[string]Kind{"a.md": Markdown, "a.log": Text, "a.go": Registered, "b.epub": Derived, "c.pdf": Derived, "d.yml": YAML} {
		if got := r.Of(p); got != want {
			t.Errorf("Of(%s) = %v, want %v", p, got, want)
		}
	}
	text = nil
	if r.Of("a.log") != Markdown {
		t.Error("a text extension the workspace dropped is still read as text")
	}
	if (Resolver{}).Of("a.epub") != Markdown {
		t.Error("a format no build derives is derived")
	}
}

// Every kind has its traits, and they are what each kind is read as.
func TestEveryKindHasItsTraits(t *testing.T) {
	want := map[Kind]Traits{
		Markdown:   {Frontmatter: true, ReadsAs: Markdown},
		Text:       {Text: true, ReadsAs: Text},
		YAML:       {Text: true, ReadsAs: YAML},
		Derived:    {ReadsAs: Markdown},
		Registered: {Text: true, ReadsAs: Registered},
	}
	if len(want) != int(numKinds) {
		t.Fatalf("the test names %d kinds; there are %d", len(want), numKinds)
	}
	for k, w := range want {
		if got := k.Traits(); got != w {
			t.Errorf("%s: %+v, want %+v", k, got, w)
		}
		if k.IsText() != w.Text {
			t.Errorf("%s: IsText %v", k, k.IsText())
		}
	}
}

// A table holds one entry per kind; one lacking a kind, or naming one that is none, is refused.
func TestATableHasEveryKind(t *testing.T) {
	all := map[Kind]string{Markdown: "m", Text: "t", YAML: "y", Derived: "d", Registered: "r"}
	tb := NewTable(all)
	for k, v := range all {
		if tb.Of(k) != v {
			t.Errorf("Of(%s) = %q", k, tb.Of(k))
		}
	}
	refused := func(entries map[Kind]string, says string) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), says) {
				t.Errorf("want a refusal saying %q, got %v", says, r)
			}
		}()
		NewTable(entries)
	}
	refused(map[Kind]string{Markdown: "m"}, "lacks the kind text")
	extra := map[Kind]string{Kind(99): "?"}
	for k, v := range all {
		extra[k] = v
	}
	refused(extra, "names 6 kinds; there are 5")
}

// The workspace's dialect reads wikilinks; a derived document's does not.
func TestTheTwoMarkdownDialects(t *testing.T) {
	src := []byte("see [[Other note]]\n")
	links := func(d *markdown.Document) int {
		n := 0
		var walk func(*markdown.Node)
		walk = func(x *markdown.Node) {
			for c := x.FirstChild; c != nil; c = c.Next {
				if c.Kind == markdown.KindWikilink {
					n++
				}
				walk(c)
			}
		}
		walk(d.Root)
		return n
	}
	if links(ParseMarkdown(src)) != 1 || links(ParseDerived(src)) != 0 {
		t.Errorf("wikilinks: workspace %d, derived %d; want 1 and 0", links(ParseMarkdown(src)), links(ParseDerived(src)))
	}
}

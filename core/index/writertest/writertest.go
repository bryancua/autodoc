// Package writertest is the conformance suite of core/index's writer port: what every place the
// indexer's documents are kept must do, run on the store's own writer and on a destination's
// (core/pgstore/pgwriter). A field an implementation does not keep yet is declared, never dropped
// silently: the suite checks that what it does not declare, it keeps.
//
// Concurrency: Run runs its cases one after another, each on a fresh writer from the harness.
package writertest

import (
	"context"
	"reflect"
	"slices"
	"testing"

	"github.com/yongjohnlee80/golib/search"
	"github.com/yongjohnlee80/golib/search/chunk"
	"github.com/yongjohnlee80/golib/vfs"

	"github.com/yongjohnlee80/autodoc/core/index"
	"github.com/yongjohnlee80/autodoc/core/schema"
)

// The fields an implementation may defer, by name.
const (
	Aliases     = "aliases"
	Diagnostics = "diagnostics"
	ADR         = "adr"
	Relations   = "relations" // links with keys: a relation's names
)

// Kept is what an implementation keeps of a document, as its harness reads it back.
type Kept struct {
	Version, Indexer, Title string
	Tags, Aliases           []string
	Facets                  map[string][]string
	Diagnostics             int
	ADR                     string
	Links                   []string // the links' names, in order
	Relations               int      // links that carry keys
	Chunks                  []KeptChunk
}

// KeptChunk is one kept chunk: in order, its text and its text's hash.
type KeptChunk struct {
	Breadcrumb, Body string
	TextHash         []byte
}

// Harness is an implementation under test.
type Harness struct {
	// New is a writer over an empty store, and how to read back what it keeps.
	New func(t *testing.T) (index.Writer, func(ctx context.Context, path string) (Kept, bool, error))
	// Defers are the fields it does not keep yet.
	Defers []string
}

// doc is a document carrying every field.
func doc(path, version, body string) index.Doc {
	c := search.Chunk{Breadcrumb: "Title", Body: body, Ord: 0}
	return index.Doc{Path: path, Version: vfs.Version(version), Indexer: "m1.t512.s0." + version,
		Meta:   chunk.Meta{Title: "Title", Tags: []string{"alpha", "beta"}, Aliases: []string{"other name"}},
		Chunks: []index.DocChunk{{Chunk: c, Hash: []byte("h-" + body), TextHash: []byte("t-" + body), Kind: index.ChunkSection}},
		Links: []index.DocLink{{Raw: "[[b]]", Name: "b", Kind: index.LinkWikilink},
			{Raw: "supersedes: c", Name: "c", Kind: "supersedes", Keys: []string{"c", "c.md"}}},
		Frontmatter: schema.Result{Facets: []schema.Facet{{Field: "status", Value: "active"}},
			Diagnostics: []schema.Diagnostic{{Field: "type", Line: 2, Rule: "required", Message: "type is required"}}},
		ADR: "0042"}
}

// Run runs the suite on h.
func Run(t *testing.T, h Harness) {
	ctx := context.Background()
	defers := func(f string) bool { return slices.Contains(h.Defers, f) }
	commit := func(t *testing.T, w index.Writer, ops func(index.Batch) error) {
		t.Helper()
		b, err := w.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := ops(b); err != nil {
			_ = b.Rollback()
			t.Fatal(err)
		}
		if err := b.Commit(); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("a put keeps every field it does not declare deferred", func(t *testing.T) {
		w, read := h.New(t)
		d := doc("a.md", "v1", "first text")
		commit(t, w, func(b index.Batch) error { return b.Put(ctx, d) })
		k, ok, err := read(ctx, "a.md")
		if err != nil || !ok {
			t.Fatalf("read back: %v %v", ok, err)
		}
		if k.Version != "v1" || k.Indexer != d.Indexer || k.Title != "Title" {
			t.Errorf("version %q, indexer %q, title %q: kept verbatim?", k.Version, k.Indexer, k.Title)
		}
		if !reflect.DeepEqual(k.Tags, []string{"alpha", "beta"}) || !reflect.DeepEqual(k.Facets, map[string][]string{"status": {"active"}}) {
			t.Errorf("tags %v, facets %v", k.Tags, k.Facets)
		}
		if len(k.Chunks) != 1 || k.Chunks[0].Body != "first text" || string(k.Chunks[0].TextHash) != "t-first text" {
			t.Errorf("chunks %+v", k.Chunks)
		}
		if len(k.Links) == 0 || k.Links[0] != "b" {
			t.Errorf("links %v", k.Links)
		}
		for f, kept := range map[string]bool{Aliases: reflect.DeepEqual(k.Aliases, []string{"other name"}),
			Diagnostics: k.Diagnostics == 1, ADR: k.ADR == "0042", Relations: k.Relations == 1} {
			if !defers(f) && !kept {
				t.Errorf("%s is neither kept nor declared deferred: %+v", f, k)
			}
		}
	})

	t.Run("a put replaces what the path kept", func(t *testing.T) {
		w, read := h.New(t)
		commit(t, w, func(b index.Batch) error { return b.Put(ctx, doc("a.md", "v1", "first text")) })
		commit(t, w, func(b index.Batch) error { return b.Put(ctx, doc("a.md", "v2", "second text")) })
		k, ok, err := read(ctx, "a.md")
		if err != nil || !ok || k.Version != "v2" || len(k.Chunks) != 1 || k.Chunks[0].Body != "second text" {
			t.Errorf("after the replace: %+v %v %v", k, ok, err)
		}
	})

	t.Run("a delete removes the path; a path not kept is no error", func(t *testing.T) {
		w, read := h.New(t)
		commit(t, w, func(b index.Batch) error { return b.Put(ctx, doc("a.md", "v1", "first text")) })
		commit(t, w, func(b index.Batch) error {
			if err := b.Delete(ctx, "a.md"); err != nil {
				return err
			}
			return b.Delete(ctx, "never.md")
		})
		if _, ok, err := read(ctx, "a.md"); ok || err != nil {
			t.Errorf("after the delete: %v %v", ok, err)
		}
	})

	t.Run("a batch is kept whole or not at all", func(t *testing.T) {
		w, read := h.New(t)
		b, err := w.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if err := b.Put(ctx, doc("a.md", "v1", "first text")); err != nil {
			t.Fatal(err)
		}
		if err := b.Rollback(); err != nil {
			t.Fatal(err)
		}
		if _, ok, err := read(ctx, "a.md"); ok || err != nil {
			t.Errorf("a rolled-back batch kept its document: %v %v", ok, err)
		}
		commit(t, w, func(b index.Batch) error {
			if err := b.Put(ctx, doc("a.md", "v1", "first text")); err != nil {
				return err
			}
			return b.Put(ctx, doc("b.md", "v1", "other text"))
		})
		for _, p := range []string{"a.md", "b.md"} {
			if _, ok, err := read(ctx, p); !ok || err != nil {
				t.Errorf("%s after a committed batch: %v %v", p, ok, err)
			}
		}
	})
}

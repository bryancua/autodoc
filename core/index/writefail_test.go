package index

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/yongjohnlee80/golib/search"

	"github.com/yongjohnlee80/autodoc/core/schema"
	"github.com/yongjohnlee80/autodoc/core/store"
)

// failWrites makes every op (INSERT, UPDATE or DELETE) on table fail, as a full disk or a
// corrupted page would; undo lifts it.
func (e *env) failWrites(table, op string) (undo func()) {
	e.t.Helper()
	name := fmt.Sprintf("fail_%s_%s", table, strings.ToLower(op))
	e.exec(fmt.Sprintf(`CREATE TRIGGER %s BEFORE %s ON "%s" BEGIN SELECT RAISE(ABORT, 'injected failure'); END`, name, op, table))
	return func() { e.exec("DROP TRIGGER " + name) }
}

// renamed moves table aside, so every read of it fails; undo puts it back.
func (e *env) renamed(table string) (undo func()) {
	e.t.Helper()
	e.exec(fmt.Sprintf(`ALTER TABLE "%s" RENAME TO "%s_aside"`, table, table))
	return func() { e.exec(fmt.Sprintf(`ALTER TABLE "%s_aside" RENAME TO "%s"`, table, table)) }
}

// TestAFailedCommitWritesNothingAndIsRetried: when the writer's transaction fails (any table's
// write refused), nothing of the batch is written, and its jobs run again: once the failure lifts,
// every change lands.
func TestAFailedCommitWritesNothingAndIsRetried(t *testing.T) {
	var failed atomic.Int64 // commits the injected failure refused
	commitFailed = func(err error) {
		if strings.Contains(err.Error(), "injected failure") {
			failed.Add(1)
		}
	}
	t.Cleanup(func() { commitFailed = nil }) // after the env's own cleanup has stopped the indexer
	e := newEnv(t, options{})
	e.put("keep.md", "---\ntags: [old]\n---\n# Keep\n\nfirst words [[other]]\n", "gone.md", "to be removed\n")
	version := func(p string) string { v, _, _ := e.store.Version(context.Background(), p); return string(v) }
	for _, c := range []struct {
		name, table, op string
		act             func()
		unchanged       func() bool
		landed          func()
	}{
		{"a new document's row", "document", "INSERT",
			func() { e.write("new1.md", "fresh\n"); e.ix.Touch("new1.md") },
			func() bool { _, ok, _ := e.store.Version(context.Background(), "new1.md"); return !ok },
			func() { e.indexedAt("new1.md") }},
		{"a new document's chunks", "chunk", "INSERT",
			func() { e.write("new2.md", "fresh words\n"); e.ix.Touch("new2.md") },
			func() bool { _, ok, _ := e.store.Version(context.Background(), "new2.md"); return !ok },
			func() { e.indexedAt("new2.md") }},
		{"a changed document's row", "document", "UPDATE",
			func() { e.write("keep.md", "# Keep\n\nsecond words\n"); e.ix.Touch("keep.md") },
			func() bool { return len(e.match("second")) == 0 },
			func() { e.indexedAt("keep.md") }},
		{"its old chunks' end", "chunk", "UPDATE",
			func() { e.write("keep.md", "# Keep\n\nthird words\n"); e.ix.Touch("keep.md") },
			func() bool { return len(e.match("third")) == 0 },
			func() { e.indexedAt("keep.md") }},
		{"its tags", "doc_tag", "INSERT",
			func() { e.write("keep.md", "---\ntags: [fresh]\n---\n# Keep\n\nfourth words\n"); e.ix.Touch("keep.md") },
			func() bool { return len(e.match("fourth")) == 0 },
			func() { e.indexedAt("keep.md") }},
		{"its links", "link", "INSERT",
			func() { e.write("keep.md", "# Keep\n\nfifth words [[gone]]\n"); e.ix.Touch("keep.md") },
			func() bool { return len(e.match("fifth")) == 0 },
			func() { e.indexedAt("keep.md") }},
		{"the change log", "change", "INSERT",
			func() { e.write("new3.md", "logged\n"); e.ix.Touch("new3.md") },
			func() bool { _, ok, _ := e.store.Version(context.Background(), "new3.md"); return !ok },
			func() { e.indexedAt("new3.md") }},
		{"the workspace's counters", "workspace", "UPDATE",
			func() { e.write("new4.md", "counted\n"); e.ix.Touch("new4.md") },
			func() bool { _, ok, _ := e.store.Version(context.Background(), "new4.md"); return !ok },
			func() { e.indexedAt("new4.md") }},
		{"a removed document's row", "document", "DELETE",
			func() { _ = e.fsys.Remove(context.Background(), "gone.md"); e.ix.Touch("gone.md") },
			func() bool { return version("gone.md") != "" },
			func() { e.eventually("gone.md gone", func() bool { return version("gone.md") == "" }) }},
	} {
		undo := e.failWrites(c.table, c.op)
		before := failed.Load()
		c.act()
		e.eventually(c.name+": a commit refused", func() bool { return failed.Load() > before })
		if !c.unchanged() {
			undo()
			t.Fatalf("%s: a failed commit wrote part of its batch", c.name)
		}
		undo()
		c.landed()
	}
}

// TestAFailedVectorCommitIsRetried: vectors whose commit fails are not stored, and the document
// stays not ready; once the failure lifts, the worker embeds again and the document becomes ready.
func TestAFailedVectorCommitIsRetried(t *testing.T) {
	p := newFake("m", "a")
	e := newEnv(t, options{Provider: p})
	e.put("a.md", "zebra plains\n")
	e.ready()
	undo := e.failWrites("embedding", "INSERT")
	e.put("b.md", "hippo river\n")
	e.eventually("b.md's text asked for", func() bool {
		for _, s := range p.texts() {
			if strings.Contains(s, "hippo river") {
				return true
			}
		}
		return false
	})
	time.Sleep(100 * time.Millisecond)
	var ready int
	_ = scanOne(context.Background(), e.raw, &ready, "SELECT semantic_ready FROM document WHERE path = 'b.md'")
	if ready != 0 {
		t.Fatal("b.md is ready though its vectors were never stored")
	}
	undo()
	e.ready()
}

// TestTheStatusAndTheViewFailAsThemselves: a read that fails in a status or a search comes back
// as an error, not as an empty answer.
func TestTheStatusAndTheViewFailAsThemselves(t *testing.T) {
	ctx := context.Background()
	p := newFake("m", "a")
	e := newEnv(t, options{Provider: p})
	e.put("a.md", "zebra plains\n")
	e.ready()
	e.atHead()
	for _, table := range []string{"change", "chunk"} {
		undo := e.renamed(table)
		_, err := e.store.Status(ctx)
		undo()
		if err == nil {
			t.Errorf("the status read with %s gone", table)
		}
	}
	st, model, vec := searchStore{s: e.store, sem: e.ix.sem}, e.activeModel(), queryVec(t, p, "a\nzebra plains")
	for _, c := range []struct {
		table string
		read  func(v *View) error
	}{
		{"chunk_fts", func(v *View) error {
			_, err := v.Lexical(ctx, []search.Term{{Text: "zebra"}}, search.Filter{}, 5)
			return err
		}},
		{"document", func(v *View) error { _, err := v.SemanticState(ctx); return err }},
		{"model", func(v *View) error { _, err := v.Semantic(ctx, model, vec, search.Filter{}, 5); return err }},
	} {
		undo := e.renamed(c.table)
		err := st.View(ctx, c.read)
		undo()
		if err == nil {
			t.Errorf("a read with %s gone answered", c.table)
		}
	}
}

// TestASearchFailsWhenItsSettingsCannotBeRead: the workspace's retrieval settings are a read of
// their own; its failing fails the search, rather than answering as if nothing were set.
func TestASearchFailsWhenItsSettingsCannotBeRead(t *testing.T) {
	e := newEnv(t, options{})
	e.put("a.md", "# A\n\nkestrel\n")
	undo := e.renamed("workspace")
	_, err := e.store.Search(context.Background(), "kestrel", QueryOpts{Limit: 5, Mode: ModeLexical})
	undo()
	if err == nil {
		t.Error("a search with the workspace table gone answered")
	}
}

// TestASweepOfOrphansFailsAsItsStatements: each read and write a sweep of orphaned vectors makes
// (the models, the vectors, the chunks that keep them, the delete) fails as an error, and the
// vectors stay; once the failure lifts, the sweep deletes them. A sweep asked before the indexer
// is ready, its request gone, returns that.
func TestASweepOfOrphansFailsAsItsStatements(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t, options{Provider: newFake("m", "a")})
	e.put("a.md", "zebra plains\n")
	e.ready()
	e.exec("DELETE FROM chunk") // every vector an orphan; the indexer, idle, writes nothing more
	vectors := func() (n int) {
		t.Helper()
		if err := scanOne(ctx, e.raw, &n, "SELECT count(*) FROM embedding"); err != nil {
			t.Fatal(err)
		}
		return n
	}
	before := vectors()
	if before == 0 {
		t.Fatal("no vectors to sweep")
	}
	for _, fail := range []struct {
		what string
		undo func() func()
	}{
		{"the models unreadable", func() func() { return e.renamed("model") }},
		{"the vectors unreadable", func() func() { return e.renamed("embedding") }},
		{"the chunks unreadable", func() func() { return e.renamed("chunk") }},
		{"the delete refused", func() func() { return e.failWrites("embedding", "DELETE") }},
	} {
		undo := fail.undo()
		if _, err := SweepOrphansStore(ctx, e.db, e.ws); err == nil {
			t.Errorf("%s: the sweep succeeded", fail.what)
		}
		undo()
		if n := vectors(); n != before {
			t.Errorf("%s: %d vectors left of %d", fail.what, n, before)
		}
	}
	// the delete path's own check, in the transaction that deleted the chunks, and the vector
	// write's: a chunk table it cannot read is an error
	undo := e.renamed("chunk")
	err := e.db.Write(ctx, func(tx *store.Tx) error {
		if _, err := e.store.dropOrphans(tx, [][]byte{{1}}); err == nil {
			t.Error("dropOrphans read no chunks and said nothing")
		}
		if _, err := e.store.withChunks(tx, []vecItem{{textHash: []byte{1}}}); err == nil {
			t.Error("withChunks read no chunks and said nothing")
		}
		return nil
	})
	undo()
	if err != nil {
		t.Fatal(err)
	}
	if n, err := SweepOrphansStore(ctx, e.db, e.ws); err != nil || n != before || vectors() != 0 {
		t.Fatalf("the sweep after: %d, %v; %d vectors left; want %d deleted", n, err, vectors(), before)
	}
	gone, cancel := context.WithCancel(ctx)
	cancel()
	if _, err := newIndexer(e.store, e.fsys, options{}).SweepOrphans(gone); err == nil {
		t.Error("a sweep of an indexer never ready, its request gone, returned no error")
	}
}

// TestAFrontmatterWriteFailsAsItsStatements: replacing a document's facets and diagnostics fails
// as whichever of its deletes and inserts is refused.
func TestAFrontmatterWriteFailsAsItsStatements(t *testing.T) {
	ctx := context.Background()
	e := newEnv(t, options{})
	e.put("a.md", "zebra\n")
	e.stop() // the store's own writes from here on: the indexer is done with a.md
	db, err := store.Open(ctx, e.db.Path())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	s := NewStore(db, e.ws)
	r := schema.Result{Facets: []schema.Facet{{Field: "status", Value: "active"}},
		Diagnostics: []schema.Diagnostic{{Field: "type", Line: 2, Rule: "required", Message: "type is required"}}}
	var docID int64
	if err := scanOne(ctx, e.raw, &docID, "SELECT id FROM document WHERE path = 'a.md'"); err != nil {
		t.Fatal(err)
	}
	replace := func() error {
		return db.Write(ctx, func(tx *store.Tx) error { return s.replaceFrontmatter(tx, docID, r) })
	}
	if err := replace(); err != nil { // the instrument: unrefused, the replacement lands
		t.Fatal(err)
	}
	for _, op := range []struct{ table, op string }{
		{"doc_facet", "DELETE"}, {"doc_diagnostic", "DELETE"}, {"doc_facet", "INSERT"}, {"doc_diagnostic", "INSERT"},
	} {
		undo := e.failWrites(op.table, op.op)
		if err := replace(); err == nil {
			t.Errorf("%s on %s refused, and the replacement succeeded", op.op, op.table)
		}
		undo()
	}
}

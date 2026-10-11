package index_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/yongjohnlee80/golib/dao"
	"github.com/yongjohnlee80/golib/dao/sqlite"

	"github.com/yongjohnlee80/autodoc/core/index"
	"github.com/yongjohnlee80/autodoc/core/index/writertest"
	"github.com/yongjohnlee80/autodoc/core/store"
)

// The store's own writer conforms, and defers nothing.
func TestTheStoresWriterConforms(t *testing.T) {
	writertest.Run(t, writertest.Harness{New: func(t *testing.T) (index.Writer, func(context.Context, string) (writertest.Kept, bool, error)) {
		ctx := context.Background()
		db, err := store.Open(ctx, filepath.Join(t.TempDir(), "autodoc.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = db.Close() })
		w, err := db.AddWorkspace(ctx, "kb", "/kb", nil, nil)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := sqlite.Open(ctx, "file:"+db.Path()+"?_pragma=busy_timeout(5000)")
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = raw.Close() })
		return index.BufferedWriter(index.NewStore(db, w.ID)), func(ctx context.Context, path string) (writertest.Kept, bool, error) {
			return readKept(ctx, raw, w.ID, path)
		}
	}})
}

// readKept reads back what the store keeps of path, with SQL of the test's own.
func readKept(ctx context.Context, raw dao.DataConn, ws int64, path string) (writertest.Kept, bool, error) {
	var k writertest.Kept
	var id, gen int64
	var title sql.NullString
	found, err := one(ctx, raw, `SELECT id, active_gen, version, indexer, title FROM document WHERE workspace_id = ? AND path = ?`,
		[]any{ws, path}, &id, &gen, &k.Version, &k.Indexer, &title)
	if err != nil || !found {
		return k, false, err
	}
	k.Title = title.String
	strs := func(q string, args ...any) ([]string, error) {
		rows, err := raw.QueryContext(ctx, q, args...)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		var out []string
		for rows.Next() {
			var s string
			if err := rows.Scan(&s); err != nil {
				return nil, err
			}
			out = append(out, s)
		}
		return out, rows.Err()
	}
	if k.Tags, err = strs(`SELECT tag FROM doc_tag WHERE doc_id = ? ORDER BY tag`, id); err != nil {
		return k, false, err
	}
	if k.Aliases, err = strs(`SELECT alias FROM doc_alias WHERE doc_id = ? ORDER BY alias`, id); err != nil {
		return k, false, err
	}
	if k.Links, err = strs(`SELECT name FROM link WHERE src_doc = ? AND gen_from <= ? AND (gen_to IS NULL OR gen_to > ?) ORDER BY id`, id, gen, gen); err != nil {
		return k, false, err
	}
	facets, err := raw.QueryContext(ctx, `SELECT field, value FROM doc_facet WHERE doc_id = ? ORDER BY field, value`, id)
	if err != nil {
		return k, false, err
	}
	for facets.Next() {
		var f, v string
		if err := facets.Scan(&f, &v); err != nil {
			return k, false, err
		}
		if k.Facets == nil {
			k.Facets = map[string][]string{}
		}
		k.Facets[f] = append(k.Facets[f], v)
	}
	_ = facets.Close()
	counts := []struct {
		q   string
		dst *int
	}{
		{`SELECT COUNT(*) FROM doc_diagnostic WHERE doc_id = ?`, &k.Diagnostics},
		{`SELECT COUNT(DISTINCT link_id) FROM link_key WHERE link_id IN (SELECT id FROM link WHERE src_doc = ?)`, &k.Relations},
	}
	for _, c := range counts {
		if _, err := one(ctx, raw, c.q, []any{id}, c.dst); err != nil {
			return k, false, err
		}
	}
	adr, err := strs(`SELECT name_key FROM doc_name WHERE doc_id = ? AND name_key LIKE '#adr:%'`, id)
	if err != nil {
		return k, false, err
	}
	if len(adr) == 1 {
		k.ADR = "00" + adr[0][len("#adr:"):] // the key drops the zeros the number was written with
	}
	chunks, err := raw.QueryContext(ctx, `SELECT breadcrumb, body, text_hash FROM chunk
		WHERE doc_id = ? AND gen_from <= ? AND (gen_to IS NULL OR gen_to > ?) ORDER BY ord`, id, gen, gen)
	if err != nil {
		return k, false, err
	}
	defer chunks.Close()
	for chunks.Next() {
		var c writertest.KeptChunk
		if err := chunks.Scan(&c.Breadcrumb, &c.Body, &c.TextHash); err != nil {
			return k, false, err
		}
		k.Chunks = append(k.Chunks, c)
	}
	return k, true, chunks.Err()
}

// one scans the first row q answers into dst, and says whether there was one.
func one(ctx context.Context, raw dao.DataConn, q string, args []any, dst ...any) (bool, error) {
	rows, err := raw.QueryContext(ctx, q, args...)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return false, rows.Err()
	}
	return true, rows.Scan(dst...)
}

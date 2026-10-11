package store

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/yongjohnlee80/golib/dao"
)

// probe is a declared set: each build records the connection it was built on.
type probe struct{ conn dao.DataConn }

var probes = Declare("probe", func(c dao.DataConn) *probe { return &probe{conn: c} })

// A declared set is built for each store's writer and for its readers, each store its own; a
// declaration after a store opened, or a second of one name, is refused.
func TestDeclaredTablesAreEachStoresOwn(t *testing.T) {
	ctx := context.Background()
	open := func() *Store {
		s, err := Open(ctx, filepath.Join(t.TempDir(), "autodoc.db"))
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { _ = s.Close() })
		return s
	}
	a, b := open(), open()
	of := func(s *Store, write bool) *probe {
		var p *probe
		run := s.Read
		if write {
			run = s.Write
		}
		if err := run(ctx, func(tx *Tx) error { p = probes.In(tx); return nil }); err != nil {
			t.Fatal(err)
		}
		return p
	}
	if of(a, true).conn != a.w || of(a, false).conn != a.r {
		t.Error("a store's declared tables are not built on its own writer and readers")
	}
	if of(a, true) == of(b, true) {
		t.Error("two stores share one declared set")
	}
	refused := func(what, says string, fn func()) {
		t.Helper()
		defer func() {
			if r := recover(); r == nil || !strings.Contains(r.(string), says) {
				t.Errorf("%s: %v, want a refusal saying %q", what, r, says)
			}
		}()
		fn()
	}
	refused("late", "declared after a store opened", func() { Declare("late", func(dao.DataConn) int { return 0 }) })
	declarations.mu.Lock()
	declarations.frozen = false // as at init, for the duplicate's check
	declarations.mu.Unlock()
	t.Cleanup(func() { declarations.mu.Lock(); declarations.frozen = true; declarations.mu.Unlock() })
	refused("twice", "declared twice", func() { Declare("probe", func(dao.DataConn) int { return 0 }) })
}

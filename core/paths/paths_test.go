package paths_test

import (
	"testing"

	"github.com/yongjohnlee80/autodoc/core/paths"
)

func TestUnder(t *testing.T) {
	for _, c := range []struct {
		p, dir string
		want   bool
	}{
		{"a/b", "a", true}, {"a", "a", true}, {"a/b/c", "a/b", true}, {"x", ".", true},
		{"a/bc", "a/b", false}, {"ab", "a", false}, {"a", "a/b", false},
		{"/home/u/kb/x.md", "/home/u/kb", true}, {"/home/u/kbx", "/home/u/kb", false},
	} {
		if got := paths.Under(c.p, c.dir); got != c.want {
			t.Errorf("Under(%q, %q) = %v, want %v", c.p, c.dir, got, c.want)
		}
	}
}

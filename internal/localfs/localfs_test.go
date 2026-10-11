package localfs_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/yongjohnlee80/autodoc/internal/localfs"
)

// A write replaces the file whole, and leaves nothing else beside it; a write with nowhere to go
// says so.
func TestWriteFileReplacesItWhole(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	p := filepath.Join(dir, "page.html")
	for _, content := range []string{"<p>first</p>", "<p>second</p>"} {
		if err := localfs.WriteFile(ctx, p, []byte(content)); err != nil {
			t.Fatal(err)
		}
		if got, _ := os.ReadFile(p); string(got) != content {
			t.Errorf("the file holds %q, want %q", got, content)
		}
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 1 {
		t.Errorf("the directory holds %d entries, want the file alone", len(entries))
	}
	if err := localfs.WriteFile(ctx, "", []byte("x")); err == nil {
		t.Error("a write to no path succeeded")
	}
	if err := localfs.WriteFile(ctx, filepath.Join(dir, "missing", "page.html"), []byte("x")); err == nil {
		t.Error("a write into a directory that is not there succeeded")
	}
}

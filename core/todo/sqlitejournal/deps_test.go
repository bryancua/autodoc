package sqlitejournal_test

import (
	"os/exec"
	"strings"
	"testing"
)

// This package joins core/todo and core/store, so neither imports the other: core/todo stays a
// pure package with its Journal port, and core/store knows no module's tables but through its seam.
func TestNeitherTodoNorStoreImportsTheOther(t *testing.T) {
	deps := func(pkg string) string {
		t.Helper()
		out, err := exec.Command("go", "list", "-deps", pkg).Output()
		if err != nil {
			t.Skipf("go list %s: %v", pkg, err)
		}
		return string(out)
	}
	const mod = "github.com/yongjohnlee80/autodoc/"
	if strings.Contains(deps(mod+"core/store"), mod+"core/todo\n") {
		t.Error("core/store imports core/todo")
	}
	if strings.Contains(deps(mod+"core/todo"), mod+"core/store\n") {
		t.Error("core/todo imports core/store")
	}
}

package paths

import "strings"

// Under reports whether p is dir or inside it, by whole segments: "a/bc" is not under "a/b". Every
// relative path is under ".".
func Under(p, dir string) bool { return dir == "." || p == dir || strings.HasPrefix(p, dir+"/") }

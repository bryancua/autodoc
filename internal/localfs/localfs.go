// Package localfs writes a file of the local disk the way core writes a workspace's: atomically,
// through golib's vfs/local, so a reader never sees half of it. It is for the programs' own files
// outside every workspace (an export's destination, a preview's page); core/ never imports it.
//
// Concurrency: WriteFile may be called from any goroutine; two writes of one path end with one of
// them whole.
package localfs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/yongjohnlee80/golib/vfs/local"
)

// WriteFile replaces the file at path with content, atomically: a temporary file beside it,
// renamed over it. path's directory must exist.
func WriteFile(ctx context.Context, path string, content []byte) error {
	if path == "" {
		return errors.New("localfs: no file to write")
	}
	dir, err := local.New(filepath.Dir(path))
	if err != nil {
		return fmt.Errorf("localfs: %s: %w", filepath.Dir(path), err)
	}
	_, err = dir.WriteFile(ctx, filepath.Base(path), bytes.NewReader(content))
	return errors.Join(err, dir.Close())
}

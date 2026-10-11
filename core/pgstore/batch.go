package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/yongjohnlee80/golib/dao"
	"github.com/yongjohnlee80/golib/errs"
)

// Batch is documents written in one transaction: kept together, or none of them. A Put that loses
// a compare-and-set to another writer (ErrWriteRace) reads the newer row and writes again, inside
// the batch, up to putRetries times; past that, or on any other error (a first insert two writers
// made, which PostgreSQL ends the transaction for), the batch has failed: Rollback it.
type Batch struct {
	s  *Store
	tx *dao.Transaction
}

// Begin starts a batch. Nothing reaches the database before its first write.
func (s *Store) Begin(ctx context.Context) *Batch { return &Batch{s: s, tx: dao.Begin(ctx)} }

// Put writes d inside the batch, as Store.Put does alone. It returns the document's id.
func (b *Batch) Put(ctx context.Context, d Document) (int64, error) {
	if d.Path == "" {
		return 0, errs.Wrap(errs.ErrInvalidArgument, "pgstore: a document has no path")
	}
	tags, facets := nonNil(d.Tags), nonNil(facetPairs(d.Facets))
	tagText := strings.Join(d.Tags, " ")
	for range putRetries {
		id, err := b.s.putIn(ctx, b.tx, d, tags, facets, tagText)
		if errors.Is(err, ErrWriteRace) {
			if err := ctx.Err(); err != nil {
				return 0, fmt.Errorf("pgstore: put %s: %w", d.Path, err)
			}
			continue // another writer bumped the generation: write again over the newer row
		}
		if err != nil {
			return 0, fmt.Errorf("pgstore: put %s: %w", d.Path, err)
		}
		return id, nil
	}
	return 0, fmt.Errorf("pgstore: put %s: %w", d.Path, ErrWriteRace)
}

// Delete removes the document at path inside the batch, with its chunks and links. A path not
// stored is no error.
func (b *Batch) Delete(ctx context.Context, path string) error {
	if err := b.s.rw.doc.On(b.tx, dao.WithQueryContext(ctx)).With(dTenant, b.s.tenant).With(dPath, path).Delete(); err != nil {
		return fmt.Errorf("pgstore: delete %s: %w", path, err)
	}
	return nil
}

// Commit keeps the batch's documents.
func (b *Batch) Commit() error { return b.tx.Commit() }

// Rollback keeps none of them.
func (b *Batch) Rollback() error { return b.tx.Rollback() }

package index

import (
	"context"

	"github.com/yongjohnlee80/autodoc/core/store"
)

// BufferedWriter is the store's own Writer outside the indexer, for the writer port's conformance
// suite (writer_conformance_test.go): a batch keeps its operations and replays them on the
// production writer (storeWriter) in one write transaction at Commit, under Begin's ctx. It is a
// test's alone: in production the only writer is the one bound to the indexer's commit, which
// orders documents with the jobs, the generation and readiness.
func BufferedWriter(s *Store) Writer { return bufferedWriter{s: s} }

type bufferedWriter struct{ s *Store }

func (w bufferedWriter) Begin(ctx context.Context) (Batch, error) {
	return &bufferedBatch{s: w.s, ctx: ctx}, nil
}

type bufferedBatch struct {
	s   *Store
	ctx context.Context // Begin's: the transaction's
	ops []func(context.Context, *storeBatch) error
}

func (b *bufferedBatch) Put(_ context.Context, d Doc) error {
	b.ops = append(b.ops, func(ctx context.Context, sb *storeBatch) error { return sb.Put(ctx, d) })
	return nil
}

func (b *bufferedBatch) Delete(_ context.Context, path string) error {
	b.ops = append(b.ops, func(ctx context.Context, sb *storeBatch) error { return sb.Delete(ctx, path) })
	return nil
}

func (b *bufferedBatch) Commit() error {
	ops := b.ops
	b.ops = nil
	if err := b.ctx.Err(); err != nil {
		return err
	}
	return b.s.db.Write(b.ctx, func(tx *store.Tx) error {
		sb := storeWriter{s: b.s, tx: tx, now: b.s.db.Clock().Now()}.batch()
		for _, op := range ops {
			if err := op(b.ctx, sb); err != nil {
				return err
			}
		}
		return nil
	})
}

func (b *bufferedBatch) Rollback() error { b.ops = nil; return nil }

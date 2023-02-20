package seriesgen

import (
	"context"

	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/storage"
	"golang.org/x/sync/errgroup"
)

func Append(ctx context.Context, goroutines int, appendable storage.Appendable, series storage.SeriesSet) error {
	g, gctx := errgroup.WithContext(ctx)

	workBuffer := make(chan storage.Series)
	for i := 0; i < goroutines; i++ {
		app := appendable.Appender(gctx)
		g.Go(func() error {
			var (
				s   storage.Series
				err error
				ok  bool
			)

			// Commit just once to improve time.
			// NOTE(bwplotka): Profile memory consequence of this.
			// Ignore error as Flush commits as well (is that enough?)
			defer func() { _ = app.Commit() }()

			for {
				select {
				case <-gctx.Done():
					return gctx.Err()
				case s, ok = <-workBuffer:
					if !ok {
						return nil
					}
				}

				ref := storage.SeriesRef(0)
				iter := s.Iterator()

				for iter.Next() {
					if gctx.Err() != nil {
						return gctx.Err()
					}
					t, v := iter.At()
					ref, err = app.Append(ref, s.Labels(), t, v)
					if err != nil {
						if rerr := app.Rollback(); rerr != nil {
							err = errors.Wrapf(err, "rollback failed: %v", rerr)
						}

						return errors.Wrap(err, "add sample")
					}
				}

				if err := iter.Err(); err != nil {
					if rerr := app.Rollback(); rerr != nil {
						err = errors.Wrapf(err, "rollback failed: %v", rerr)
					}
					return errors.Wrap(err, "iter")
				}
			}
		})
	}

	for series.Next() {
		select {
		// TODO(bwplotka): Add some progress bar.
		case workBuffer <- series.At():
		case <-gctx.Done():
			return errors.Wrapf(g.Wait(), "err: %s", gctx.Err())
		}

	}
	close(workBuffer)
	if series.Err() != nil {
		_ = g.Wait()
		return series.Err()
	}

	return g.Wait()
}

package blockgen

import (
	"context"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
)

// NewBlockWriter create new TSDB block writer.
//
// The returned writer is generally not assumed to be thread-safe at the moment.
//
// The returned writer accumulates all series in memory until `Flush` is called.
// The repeated pattern of writes and flushes is allowed e.g.:
//
//	for n < 1000 {
//		// write a lot of stuff into memory
//		w.Write()
//		w.Write()
//
//		// write block to disk
//		w.Flush()
//  }
//
// The above loop will produce 1000 blocks on disk.
//
// In general flushing is slow. If you have only few samples it may be much
// more efficient to flush just once the entire retention period.
//
// Note that the writer will not check if the target directory exists or
// contains anything at all. It is the caller's responsibility to
// ensure that the resulting blocks do not overlap etc.
func NewBlockWriter(logger log.Logger, dir string) (Writer, error) {
	res := &blockWriter{
		logger: logger,
		dir:    dir,
	}

	if err := res.initHeadAndAppender(); err != nil {
		return nil, err
	}

	return res, nil
}

// blockWriter is implementation of Writer interface. Not designed to be thread-safe.
type blockWriter struct {
	// logger is given to us as arg.
	logger log.Logger

	// dir is output directory, given to us as arg.
	dir string

	// prometheus specific things, created and managed by us.
	head     *tsdb.Head
	appender tsdb.Appender

	// metricCount is incremented internally every time Write is called.
	metricCount int64
}

// Write implements Writer interface. Everything goes into memory until Flush.
func (w *blockWriter) Write(t time.Time, v Val) error {
	// Simply write to appender until Flush() is called.
	w.metricCount++

	if _, err := w.appender.Add(v.Labels(), timestamp.FromTime(t), v.Val()); err != nil {
		return errors.Wrap(err, "appender.Add")
	}

	return nil
}

// Flush implements Writer interface. This is where actual block writing
// happens. After flush completes, more writes can continue.
func (w *blockWriter) Flush() error {
	// Flush should:
	//  - write head to disk
	//  - close head
	//  - open new head and appender
	if err := w.writeHeadToDisk(); err != nil {
		return errors.Wrap(err, "writeHeadToDisk")
	}

	if err := w.head.Close(); err != nil {
		return errors.Wrap(err, "close head")
	}

	if err := w.initHeadAndAppender(); err != nil {
		return errors.Wrap(err, "initHeadAndAppender")
	}

	return nil
}

// initHeadAndAppender creates and initialises new head and appender.
func (w *blockWriter) initHeadAndAppender() error {
	logger := w.logger

	var head *tsdb.Head
	{
		// chunkRange determines which events are compactable.
		// setting to 1 seems to be the right thing as want all events.
		var chunkRange int64 = 1

		// Registerer and WAL can be nil as we don't use them.
		// Not declaring to avoid dependency on github.com/prometheus/client_golang
		// var r prometheus.Registerer = nil
		// var w *wal.WAL = nil

		h, err := tsdb.NewHead(nil /*Registerer*/, logger, nil /*WAL*/, chunkRange)
		if err != nil {
			return errors.Wrap(err, "tsdb.NewHead")
		}

		head = h
	}

	w.head = head
	w.appender = head.Appender()
	return nil
}

// writeHeadToDisk commits the appender and writes the head to disk.
func (w *blockWriter) writeHeadToDisk() error {
	if err := w.appender.Commit(); err != nil {
		return errors.Wrap(err, "appender.Commit")
	}

	seriesCount := w.head.NumSeries()
	mint := timestamp.Time(w.head.MinTime())
	maxt := timestamp.Time(w.head.MaxTime())
	level.Info(w.logger).Log(
		"series_count", seriesCount,
		"metric_count", w.metricCount,
		"mint", mint,
		"maxt", maxt)

	// Step 2. Flush head to disk.
	//
	// copypasta from: github.com/prometheus/prometheus/tsdb/db.go:322
	//
	// Add +1 millisecond to block maxt because block intervals are half-open: [b.MinTime, b.MaxTime).
	//
	// Because of this block intervals are always +1 than the total samples it includes.
	{
		int_mint := timestamp.FromTime(mint)
		int_maxt := timestamp.FromTime(maxt)

		// TODO(ppanyukov): what exactly is "ranges" arg here?
		compactor, err := tsdb.NewLeveledCompactor(
			context.Background(),
			nil,
			w.logger,
			tsdb.DefaultOptions.BlockRanges,
			chunkenc.NewPool())
		if err != nil {
			return errors.Wrap(err, "create leveled compactor")
		}

		_, err = compactor.Write(w.dir, w.head, int_mint, int_maxt+1, nil)
		return errors.Wrap(err, "writing WAL")
	}
}

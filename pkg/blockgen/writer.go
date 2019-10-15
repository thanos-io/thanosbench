package blockgen

import (
	"context"
	"runtime"
	"time"

	"github.com/oklog/ulid"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
)

var _ Writer = &BlockWriter{}

// BlockWriter is implementation of Writer interface. Not designed to be thread-safe.
type BlockWriter struct {
	// logger is given to us as arg.
	logger log.Logger

	// dir is output directory, given to us as arg.
	dir string

	// prometheus specific things, created and managed by us.
	head     *tsdb.Head
	appender tsdb.Appender
}

// NewTSDBBlockWriter create new TSDB block writer.
//
// The returned writer is generally not assumed to be thread-safe at the moment.
// It is assumed for single use.
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
// Note that the writer will not check if the target directory exists or
// contains anything at all. It is the caller's responsibility to
// ensure that the resulting blocks do not overlap etc.
func NewTSDBBlockWriter(logger log.Logger, dir string) (*BlockWriter, error) {
	res := &BlockWriter{
		logger: logger,
		dir:    dir,
	}

	if err := res.initHeadAndAppender(); err != nil {
		return nil, err
	}

	return res, nil
}

func (w *BlockWriter) Appender() tsdb.Appender {
	return w.head.Appender()
}

// Flush implements Writer interface. This is where actual block writing
// happens. After flush completes, no write can be done.
func (w *BlockWriter) Flush() (ulid.ULID, error) {
	id, err := w.writeHeadToDisk()
	if err != nil {
		return ulid.ULID{}, errors.Wrap(err, "writeHeadToDisk")
	}

	if err := w.head.Close(); err != nil {
		return ulid.ULID{}, errors.Wrap(err, "close head")
	}

	return id, nil
}

// initHeadAndAppender creates and initialises new head and appender.
func (w *BlockWriter) initHeadAndAppender() error {
	logger := w.logger

	var head *tsdb.Head
	{
		// chunkRange determines which events are compactable.
		// setting to 1 seems to be the right thing as want all events.
		var chunkRange int64 = 1

		// Keep Registerer and WAL nil as we don't use them.
		// Not declaring to avoid dependency on github.com/prometheus/client_golang
		//    var r prometheus.Registerer = nil
		//    var w *wal.WAL = nil
		h, err := tsdb.NewHead(nil, logger, nil, chunkRange)
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
func (w *BlockWriter) writeHeadToDisk() (ulid.ULID, error) {
	if err := w.appender.Commit(); err != nil {
		return ulid.ULID{}, errors.Wrap(err, "appender.Commit")
	}

	seriesCount := w.head.NumSeries()

	if w.head.NumSeries() == 0 {
		return ulid.ULID{}, errors.New("no series appended; aborting.")
	}

	mint := w.head.MinTime()
	maxt := w.head.MaxTime()
	level.Info(w.logger).Log(
		"msg", "flushing",
		"series_count", seriesCount,
		"mint", timestamp.Time(mint),
		"maxt", timestamp.Time(maxt),
	)
	// Flush head to disk as a block.
	compactor, err := tsdb.NewLeveledCompactor(
		context.Background(),
		nil,
		w.logger,
		[]int64{durToMilis(2 * time.Hour)}, // Does not matter, used only for planning.
		chunkenc.NewPool())
	if err != nil {
		return ulid.ULID{}, errors.Wrap(err, "create leveled compactor")
	}

	defer runtime.GC()

	return compactor.Write(w.dir, w.head, mint, maxt+1, nil)
}

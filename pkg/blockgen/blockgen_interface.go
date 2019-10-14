package blockgen

import (
	"time"

	"github.com/prometheus/prometheus/tsdb/labels"
)

// Val is the named value to write to time series db. It will be timestamped by
// the generator.
type Val interface {
	Val() float64
	Labels() labels.Labels
}

// ValProvider is the generator of synthetic values.
type ValProvider interface {
	// Next returns a chan from which provides values for one sampling interval.
	Next() <-chan Val
}

// Writer is interface to write time series into Prometheus blocks.
type Writer interface {
	// Writes one value, into memory.
	// TODO(ppanyukov): how about re-using tsdb.Appendable instead?
	Write(t time.Time, v Val) error

	// Flush writes current block to disk.
	// The block will contain values accumulated by `Write`.
	Flush() error
}

// Generator generates synthetic time series using values produced by supplied
// list of `ValProvider` and writes them to TSDB blocks using supplied `Writer`.
type Generator interface {
	Generate(writer Writer, valGenerators ...ValProvider) error
}

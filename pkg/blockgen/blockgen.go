package blockgen

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"time"
	"unsafe"

	"github.com/pkg/errors"

	"github.com/go-kit/kit/log"
	"github.com/oklog/ulid"
	promlabels "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/labels"
	"github.com/thanos-io/thanos/pkg/block/metadata"
	"github.com/thanos-io/thanosbench/pkg/seriesgen"
)

// Writer is interface to write time series into Prometheus blocks.
type Writer interface {
	tsdb.Appendable

	// Flush writes current block to disk.
	// The block will contain values accumulated by `Write`.
	Flush() (ulid.ULID, error)
}

// TODO(bwplotka): Add option to create downsampled blocks.
type BlockSpec struct {
	metadata.Meta
	Series []SeriesSpec
}

type GenType string

const (
	Random  GenType = "RANDOM"
	Counter GenType = "COUNTER"
	Gauge   GenType = "GAUGE"
)

func (g GenType) Create(random *rand.Rand, mint, maxt int64, opts seriesgen.Characteristics) (seriesgen.SeriesIterator, error) {
	switch g {
	case Random:
		return seriesgen.NewValGen(random, mint, maxt, opts), nil
	case Counter:
		return seriesgen.NewCounterGen(random, mint, maxt, opts), nil
	case Gauge:
		return seriesgen.NewGaugeGen(random, mint, maxt, opts), nil
	default:
		return nil, errors.Errorf("unknown type: %s", string(g))
	}
}

func toLabels(lset promlabels.Labels) labels.Labels {
	return *(*labels.Labels)(unsafe.Pointer(&lset))
}

type SeriesSpec struct {
	Labels promlabels.Labels `yaml:"labels"`

	// Targets multiples labels by given targets.
	Targets int `yaml:"targets"`

	Type GenType `yaml:"type"`

	MinTime, MaxTime int64

	seriesgen.Characteristics `yaml:",inline"`
}

func durToMilis(t time.Duration) int64 {
	return int64(t.Seconds() * 1000)
}

// Generate creates a block from given spec using given go routines in a given directory.
func Generate(ctx context.Context, logger log.Logger, goroutines int, dir string, block BlockSpec) (ulid.ULID, error) {
	w, err := NewTSDBBlockWriter(logger, dir)
	if err != nil {
		return ulid.ULID{}, err
	}
	set := &blockSeriesSet{config: block}
	if err := seriesgen.Append(ctx, goroutines, w, set); err != nil {
		return ulid.ULID{}, errors.Wrap(err, "append")
	}
	id, err := w.Flush()
	if err != nil {
		return ulid.ULID{}, errors.Wrap(err, "flush")
	}

	bdir := path.Join(dir, id.String())
	meta, err := metadata.Read(bdir)
	if err != nil {
		return ulid.ULID{}, errors.Wrap(err, "meta read")
	}
	meta.Thanos = block.Thanos
	if err := metadata.Write(logger, bdir, meta); err != nil {
		return ulid.ULID{}, errors.Wrap(err, "meta write")
	}
	return id, nil
}

type blockSeriesSet struct {
	config BlockSpec
	i      int
	target int
	err    error

	curr seriesgen.Series
}

func (s *blockSeriesSet) Next() bool {
	if s.target > 0 {
		s.target--
	}
	if s.target <= 0 && s.i >= len(s.config.Series) {
		return false
	}

	if s.target <= 0 {
		s.i++
		s.target = s.config.Series[s.i-1].Targets
	}

	series := s.config.Series[s.i-1]
	lset := labels.Labels(append([]labels.Label{{Name: "__blockgen_target__", Value: fmt.Sprintf("%v", s.target)}}, toLabels(series.Labels)...))

	// Stable random per series name.
	iter, err := series.Type.Create(
		rand.New(rand.NewSource(int64(lset.Hash()))),
		series.MinTime,
		series.MaxTime,
		series.Characteristics,
	)
	if err != nil {
		s.err = err
		return false
	}
	s.curr = seriesgen.NewSeriesGen(lset, iter)
	return true
}

func (s *blockSeriesSet) At() seriesgen.Series { return s.curr }

func (s *blockSeriesSet) Err() error { return s.err }

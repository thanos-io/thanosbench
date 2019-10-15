package blockgen

import (
	"context"
	"fmt"
	"math/rand"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v2"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/ulid"
	"github.com/pkg/errors"
	promlabels "github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
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

type GeneratorConfig struct {
	RangeBasedBlocks []RangesBasedConfig `yaml:"rangeBasedBlocks"`
	Blocks           []BlockConfig       `yaml:"blocks"`
}

type RangesBasedConfig struct {
	// StartTime is the time from which to generate metrics. The metrics are not strictly generated from this date (!).
	// It aligned to the 2h range as on Prometheus.
	StartTime time.Time `yaml:"startTime"`
	// From newest to oldest. Retention of the metric will be sum of all blocks.
	Blocks []time.Duration `yaml:"blocks"`
	// Series across all blocks.
	Series         []SeriesConfig    `yaml:"series"`
	ExternalLabels promlabels.Labels `yaml:"externalLabels"`
}

// TODO(bwplotka): Add option to create downsampled blocks.
type BlockConfig struct {
	metadata.Meta
	Series []SeriesConfig
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
	// TODO(bwplotka): Make it efficient or write marshsaller.
	return labels.FromMap(lset.Map())
}

// TODO(bwplotka): Allow partial block series.
type SeriesConfig struct {
	Labels promlabels.Labels `yaml:"labels"`
	// Targets multiples labels by given targets.
	Targets int `yaml:"targets"`

	Type                      GenType `yaml:"type"`
	seriesgen.Characteristics `yaml:",inline"`
}

func LoadConfig(cfg []byte) (GeneratorConfig, error) {
	g := GeneratorConfig{}
	if err := yaml.UnmarshalStrict(cfg, &g); err != nil {
		return GeneratorConfig{}, err
	}

	return g, nil
}

type NoopSyncer struct{}

func (NoopSyncer) Sync(ctx context.Context, bdir string) error { return nil }

type Syncer interface {
	Sync(ctx context.Context, bdir string) error
}

func Generate(ctx context.Context, logger log.Logger, dir string, workersNum int, syncer Syncer, cfg GeneratorConfig) error {
	var toGenerate []BlockConfig
	for _, c := range cfg.RangeBasedBlocks {
		if len(c.Blocks) == 0 {
			return errors.New("empty block ranges")
		}

		if c.StartTime.Equal(time.Unix(0, 0)) {
			c.StartTime = time.Now()
		}

		maxt := rangeForTimestamp(timestamp.FromTime(c.StartTime), durToMilis(2*time.Hour))
		for _, r := range c.Blocks {
			mint := maxt - durToMilis(r)
			toGenerate = append(toGenerate, BlockConfig{
				Meta: metadata.Meta{
					BlockMeta: tsdb.BlockMeta{
						MaxTime: maxt,
						MinTime: mint,
						// TODO(bwplotka): Allow customization.
						Compaction: tsdb.BlockMetaCompaction{
							Level: 0,
						},
					},
					Thanos: metadata.Thanos{
						Labels: c.ExternalLabels.Map(),
						Downsample: metadata.ThanosDownsample{
							Resolution: 0,
						},
						Source: "blockgen",
					},
				},
				Series: c.Series,
			})
			maxt = mint
		}
	}

	toGenerate = append(toGenerate, cfg.Blocks...)

	level.Info(logger).Log("msg", "scheduled blocks to generate", "workers", workersNum, "blocks", printBlocks(toGenerate...), "dir", dir)

	for _, b := range toGenerate {
		level.Info(logger).Log("msg", "generating block", "blocks", printBlocks(b), "dir", dir)

		start := time.Now()
		id, err := generateBlock(ctx, logger, workersNum, dir, b)
		if err != nil {
			return errors.Wrap(err, "generate block")
		}
		bdir := path.Join(dir, id.String())
		meta, err := metadata.Read(bdir)
		if err != nil {
			return errors.Wrap(err, "meta read")
		}
		meta.Thanos = b.Thanos
		if err := metadata.Write(logger, bdir, meta); err != nil {
			return errors.Wrap(err, "meta write")
		}
		level.Info(logger).Log("msg", "generated block", "id", id.String(), "blocks", printBlocks(b), "dir", dir, "elapsed", time.Since(start))

		if err := syncer.Sync(ctx, bdir); err != nil {
			return errors.Wrap(err, "sync")
		}
	}
	return nil
}

func rangeForTimestamp(t int64, width int64) (maxt int64) {
	return (t/width)*width + width
}

func printBlocks(bts ...BlockConfig) string {
	var msg []string
	for _, b := range bts {
		msg = append(msg, fmt.Sprintf("[%d - %d](%s) ", b.MinTime, b.MaxTime, milisToDur(b.MaxTime-b.MinTime).String()))
	}
	return strings.Join(msg, ",")
}

func durToMilis(t time.Duration) int64 {
	return int64(t.Seconds() * 1000)
}

func milisToDur(t int64) time.Duration {
	return time.Duration(t * int64(time.Millisecond))
}

func generateBlock(ctx context.Context, logger log.Logger, workersNum int, dir string, block BlockConfig) (ulid.ULID, error) {
	w, err := NewTSDBBlockWriter(logger, dir)
	if err != nil {
		return ulid.ULID{}, err
	}
	set := &blockSeriesSet{config: block}
	if err := seriesgen.Append(ctx, workersNum, w, set); err != nil {
		return ulid.ULID{}, errors.Wrap(err, "append")
	}
	return w.Flush()
}

type blockSeriesSet struct {
	config BlockConfig
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
		s.config.MinTime,
		s.config.MaxTime,
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

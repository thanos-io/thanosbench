package seriesgen

import (
	"math/rand"
	"time"

	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/storage"
	"github.com/prometheus/prometheus/tsdb/chunkenc"
)

type sample struct {
	T int64
	V float64
}

type SeriesGen struct {
	chunkenc.Iterator

	lset labels.Labels
}

func NewSeriesGen(lset labels.Labels, it chunkenc.Iterator) storage.Series {
	return &storage.SeriesEntry{
		Lset: lset,
		SampleIteratorFn: func() chunkenc.Iterator {
			return it
		},
	}
}
func (s *SeriesGen) Labels() labels.Labels { return s.lset }

var _ chunkenc.Iterator = &GaugeGen{}
var _ chunkenc.Iterator = &CounterGen{}
var _ chunkenc.Iterator = &ValGen{}

type Characteristics struct {
	Jitter         float64       `yaml:"jitter"`
	ScrapeInterval time.Duration `yaml:"scrapeInterval"`
	ChangeInterval time.Duration `yaml:"changeInterval"`
	Max            float64       `yaml:"max"`
	Min            float64       `yaml:"min"`
}

type GaugeGen struct {
	changeInterval   time.Duration
	interval         time.Duration
	maxTime, minTime int64

	min, max, jitter float64

	v       float64
	mod     float64
	init    bool
	elapsed int64

	random *rand.Rand
}

func NewGaugeGen(random *rand.Rand, mint, maxt int64, opts Characteristics) *GaugeGen {
	return &GaugeGen{
		changeInterval: opts.ChangeInterval,
		interval:       opts.ScrapeInterval,
		max:            opts.Max,
		min:            opts.Min,
		minTime:        mint,
		maxTime:        maxt,
		jitter:         opts.Jitter,
		random:         random,
	}
}

func (g *GaugeGen) Next() bool {
	if g.minTime > g.maxTime {
		return false
	}
	defer func() {
		g.minTime += int64(g.interval.Seconds() * 1000)
		g.elapsed += int64(g.interval.Seconds() * 1000)
	}()

	if !g.init {
		g.v = g.min + g.random.Float64()*((g.max-g.min)+1)
		g.init = true
	}

	// Technically only mod changes.
	if g.jitter > 0 && g.elapsed >= int64(g.changeInterval.Seconds()*1000) {
		g.mod = (g.random.Float64() - 0.5) * g.jitter
		g.elapsed = 0
	}
	return true
}

func (g *GaugeGen) At() (t int64, v float64) {
	return g.minTime, g.v + g.mod
}

func (g *GaugeGen) Err() error { return nil }

func (g *GaugeGen) Seek(_ int64) bool { return true }

// TODO(bwplotka): Improve. Does not work well (: Too naive.
// Add resets etc.
type CounterGen struct {
	maxTime, minTime int64

	min, max, jitter float64
	interval         time.Duration
	changeInterval   time.Duration
	rateInterval     time.Duration

	v    float64
	init bool
	buff []sample

	lastVal float64
	elapsed int64

	random *rand.Rand
}

func NewCounterGen(random *rand.Rand, mint, maxt int64, opts Characteristics) *CounterGen {
	return &CounterGen{
		changeInterval: opts.ChangeInterval,
		interval:       opts.ScrapeInterval,
		max:            opts.Max,
		min:            opts.Min,
		minTime:        mint,
		maxTime:        maxt,
		jitter:         opts.Jitter,
		random:         random,
		rateInterval:   5 * time.Minute,
	}
}

func (g *CounterGen) Next() bool {
	defer func() { g.elapsed += int64(g.interval.Seconds() * 1000) }()

	if g.init && len(g.buff) == 0 {
		return false
	}

	if len(g.buff) > 0 {
		// Pop front.
		g.buff = g.buff[1:]

		if len(g.buff) > 0 {
			return true
		}
	}

	if !g.init {
		g.v = g.min + g.random.Float64()*((g.max-g.min)+1)
		g.init = true
	}

	var mod float64
	if g.jitter > 0 && g.elapsed >= int64(g.changeInterval.Seconds()*1000) {
		mod = (g.random.Float64() - 0.5) * g.jitter

		if mod > g.v {
			mod = g.v
		}

		g.elapsed = 0
	}

	// Distribute goalV into multiple rateInterval/interval increments.
	comps := make([]float64, int64(g.rateInterval/g.interval))
	var sum float64
	for i := range comps {
		comps[i] = g.random.Float64()
		sum += comps[i]
	}

	// That's the goal for our rate.
	x := g.v + mod/sum
	for g.minTime <= g.maxTime && len(comps) > 0 {
		g.lastVal += x * comps[0]
		comps = comps[1:]

		g.minTime += int64(g.interval.Seconds() * 1000)
		g.buff = append(g.buff, sample{T: g.minTime, V: g.lastVal})
	}

	return len(g.buff) > 0
}

func (g *CounterGen) At() (int64, float64) { return g.buff[0].T, g.buff[0].V }

func (g *CounterGen) Err() error { return nil }

func (g *CounterGen) Seek(_ int64) bool { return true }

type ValGen struct {
	interval         time.Duration
	maxTime, minTime int64

	min, max float64

	v      float64
	random *rand.Rand
}

func NewValGen(random *rand.Rand, mint, maxt int64, opts Characteristics) *ValGen {
	return &ValGen{
		interval: opts.ScrapeInterval,
		max:      opts.Max,
		min:      opts.Min,
		minTime:  mint,
		maxTime:  maxt,
		random:   random,
	}
}

func (g *ValGen) Next() bool {
	if g.minTime > g.maxTime {
		return false
	}

	g.minTime += int64(g.interval.Seconds() * 1000)
	g.v = g.min + g.random.Float64()*((g.max-g.min)+1)

	return true
}

func (g *ValGen) At() (t int64, v float64) {
	return g.minTime, g.v
}

func (g *ValGen) Err() error { return nil }

func (g *ValGen) Seek(_ int64) bool { return true }

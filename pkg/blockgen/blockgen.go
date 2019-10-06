package blockgen

import (
	"encoding/json"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/promql"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/prometheus/prometheus/tsdb/labels"
)

// TODO(bwplotka): Allow more realistic output.
type Config struct {
	InputSeries    []Series
	Retention      time.Duration
	ScrapeInterval time.Duration
}

type Series struct {
	Type           string // gauge, counter (if counter we treat below as rate aim)
	Jitter         float64
	ChangeInterval string
	Max            float64
	Min            float64
	// Result is an exact Prometheus HTTP query result that would be used to generate series' metrics labels.
	Result QueryData
	// Replicate multiples this set given number of times. For example if result has 10 metrics and replicate is 10 we will
	// have 100 unique series.
	Replicate int
}

type QueryData struct {
	ResultType model.ValueType `json:"resultType"`
	Result     model.Vector    `json:"result"`
}

func GenerateTSDB(logger log.Logger, dir string, configContent []byte) error {
	var config Config
	if err := json.Unmarshal(configContent, &config); err != nil {
		return err
	}

	if config.ScrapeInterval == 0 {
		config.ScrapeInterval = 15 * time.Second
	}

	// Same code as Prometheus for compaction levels and max block.
	rngs := tsdb.ExponentialBlockRanges(int64(time.Duration(2*time.Hour).Seconds()*1000), 10, 3)
	maxBlockDuration := config.Retention / 10
	for i, v := range rngs {
		if v > int64(maxBlockDuration.Seconds()*1000) {
			rngs = rngs[:i]
			break
		}
	}

	if len(rngs) == 0 {
		rngs = append(rngs, int64(time.Duration(2*time.Hour).Seconds()*1000))
	}

	// TODO(bwplotka): Moved to something like https://github.com/thanos-io/thanos/blob/master/pkg/testutil/prometheus.go#L289
	//  to actually generate blocks! It will be fine for TSDB use cases as well.
	db, err := tsdb.Open(dir, nil, nil, &tsdb.Options{
		BlockRanges:       rngs,
		RetentionDuration: uint64(config.Retention.Seconds() * 1000),
		NoLockfile:        true,
	})
	if err != nil {
		level.Error(logger).Log("err", err)
		os.Exit(1)
	}

	// Of course there will be small gap in minTime vs time.Now once we finish.
	// We are fine with this.
	n := time.Now()
	maxTime := timestamp.FromTime(n)
	minTime := timestamp.FromTime(n.Add(-config.Retention))

	// TODO(bwplotka): Consider sharding for reduced memory use for generating.
	// TODO(bwplotk): Do it concurrently.
	generators := make(map[string]gen)
	for _, in := range config.InputSeries {
		for _, r := range in.Result.Result {
			for i := 0; i < in.Replicate; i++ {
				lset := labels.New()
				for n, v := range r.Metric {
					lset = append(lset, labels.Label{Name: string(n), Value: string(v)})
				}
				if i > 0 {
					lset = append(lset, labels.Label{Name: "blockgen_fake_replica", Value: strconv.Itoa(i)})
				}

				var chInterval time.Duration
				if in.ChangeInterval != "" {
					chInterval, err = time.ParseDuration(in.ChangeInterval)
					if err != nil {
						return err
					}
				}

				switch strings.ToLower(in.Type) {
				case "counter":
					// Does not work well (: Too naive.
					generators[lset.String()] = &counterGen{
						interval:       config.ScrapeInterval,
						maxTime:        maxTime,
						minTime:        minTime,
						lset:           lset,
						min:            in.Min,
						max:            in.Max,
						jitter:         in.Jitter,
						rateInterval:   5 * time.Minute,
						changeInterval: chInterval,
					}
				case "gauge":
					generators[lset.String()] = &gaugeGen{
						interval:       config.ScrapeInterval,
						maxTime:        maxTime,
						minTime:        minTime,
						lset:           lset,
						min:            in.Min,
						max:            in.Max,
						jitter:         in.Jitter,
						changeInterval: chInterval,
					}
				default:
					return errors.Errorf("failed to parse series, unknown metric type: %s", in.Type)
				}
			}
		}
	}

	a := db.Appender()
	for _, generator := range generators {
		for generator.Next() {
			// Cache reference and use AddFast if we are too slow.
			if _, err := a.Add(generator.Lset(), generator.Ts(), generator.Value()); err != nil {
				return errors.Wrap(err, "add")
			}
		}
	}

	if err := a.Commit(); err != nil {
		return errors.Wrap(err, "commit")
	}

	// Don't wait for compact, it will be compacted by Prometheus anyway.

	if err := db.Close(); err != nil {
		return errors.Wrap(err, "close")
	}

	level.Info(logger).Log("msg", "generated artificial metrics", "series", len(generators))
	return nil
}

type gaugeGen struct {
	changeInterval   time.Duration
	interval         time.Duration
	maxTime, minTime int64

	lset             labels.Labels
	min, max, jitter float64

	v       float64
	mod     float64
	init    bool
	elapsed int64
}

func (g *gaugeGen) Lset() labels.Labels {
	return g.lset
}

func (g *gaugeGen) Next() bool {
	if g.minTime > g.maxTime {
		return false
	}
	defer func() {
		g.minTime += int64(g.interval.Seconds() * 1000)
		g.elapsed += int64(g.interval.Seconds() * 1000)
	}()

	if !g.init {
		g.v = g.min + rand.Float64()*((g.max-g.min)+1)
		g.init = true
	}

	// Technically only mod changes.
	if g.jitter > 0 && g.elapsed >= int64(g.changeInterval.Seconds()*1000) {
		g.mod = (rand.Float64() - 0.5) * g.jitter
		g.elapsed = 0
	}

	return true
}

func (g *gaugeGen) Ts() int64      { return g.minTime }
func (g *gaugeGen) Value() float64 { return g.v + g.mod }

type counterGen struct {
	maxTime, minTime int64

	lset             labels.Labels
	min, max, jitter float64
	interval         time.Duration
	changeInterval   time.Duration
	rateInterval     time.Duration

	v    float64
	init bool
	buff []promql.Point

	lastVal float64
	elapsed int64
}

func (g *counterGen) Lset() labels.Labels {
	return g.lset
}

func (g *counterGen) Next() bool {
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
		g.v = g.min + rand.Float64()*((g.max-g.min)+1)
		g.init = true
	}

	var mod float64
	if g.jitter > 0 && g.elapsed >= int64(g.changeInterval.Seconds()*1000) {
		mod = (rand.Float64() - 0.5) * g.jitter

		if mod > g.v {
			mod = g.v
		}

		g.elapsed = 0
	}

	// Distribute goalV into multiple rateInterval/interval increments.
	comps := make([]float64, int64(g.rateInterval/g.interval))
	var sum float64
	for i := range comps {
		comps[i] = rand.Float64()
		sum += comps[i]
	}

	// That's the goal for our rate.
	x := g.v + mod/sum
	for g.minTime <= g.maxTime && len(comps) > 0 {
		g.lastVal += x * comps[0]
		comps = comps[1:]

		g.minTime += int64(g.interval.Seconds() * 1000)
		g.buff = append(g.buff, promql.Point{T: g.minTime, V: g.lastVal})
	}

	return len(g.buff) > 0
}

func (g *counterGen) Ts() int64      { return g.buff[0].T }
func (g *counterGen) Value() float64 { return g.buff[0].V }

type gen interface {
	Lset() labels.Labels
	Next() bool
	Ts() int64
	Value() float64
}

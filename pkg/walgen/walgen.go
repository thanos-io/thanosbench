package walgen

import (
	"context"
	"encoding/json"
	"math/rand"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/thanos-io/thanosbench/pkg/seriesgen"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/pkg/errors"
	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/prometheus/prometheus/tsdb"
)

// TODO(bwplotka): Allow more realistic output.
type Config struct {
	InputSeries    []Series
	Retention      time.Duration
	ScrapeInterval time.Duration
}

type Series struct {
	Type string // gauge, counter (if counter we treat below as rate aim).

	Characteristics seriesgen.Characteristics

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

	random := rand.New(rand.NewSource(1234))

	set := &Set{}
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
				switch strings.ToLower(in.Type) {
				case "counter":
					set.s = append(set.s, seriesgen.NewSeriesGen(lset, seriesgen.NewCounterGen(random, minTime, maxTime, in.Characteristics)))
				case "gauge":
					set.s = append(set.s, seriesgen.NewSeriesGen(lset, seriesgen.NewGaugeGen(random, minTime, maxTime, in.Characteristics)))
				default:
					return errors.Errorf("failed to parse series, unknown metric type: %s", in.Type)
				}
			}
		}
	}

	if err := seriesgen.Append(context.Background(), 2*runtime.GOMAXPROCS(0), db, set); err != nil {
		return errors.Wrap(err, "commit")
	}

	// Don't wait for compact, it will be compacted by Prometheus anyway.

	if err := db.Close(); err != nil {
		return errors.Wrap(err, "close")
	}

	level.Info(logger).Log("msg", "generated artificial metrics", "series", len(set.s))
	return nil
}

type Set struct {
	s    []seriesgen.Series
	curr int
}

func (s *Set) Next() bool {
	if s.curr > len(s.s) {
		return false
	}
	s.curr++
	return true
}

func (s *Set) At() seriesgen.Series { return s.s[s.curr-1] }

func (s *Set) Err() error { return nil }

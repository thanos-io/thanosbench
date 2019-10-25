package blockgen

import (
	"context"
	"fmt"
	"time"

	"github.com/prometheus/prometheus/pkg/timestamp"

	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/prometheus/prometheus/tsdb"
	"github.com/thanos-io/thanos/pkg/block/metadata"
	"github.com/thanos-io/thanos/pkg/model"
	"github.com/thanos-io/thanosbench/pkg/seriesgen"
)

type ProfileMap map[string]func(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error

func (p ProfileMap) Keys() (keys []string) {
	for k := range p {
		keys = append(keys, k)
	}
	return keys
}

var (
	Profiles = ProfileMap{
		"realistic-k8s-1w-small": realisticK8s1w,
	}
)

func realisticK8s1w(ctx context.Context, maxTime model.TimeOrDurationValue, extLset labels.Labels, blockEncoder func(BlockSpec) error) error {
	// Align timestamps as Prometheus would do.
	maxt := rangeForTimestamp(maxTime.PrometheusTimestamp(), durToMilis(2*time.Hour))

	// Track "rollouts". In heavy used K8s we have rollouts every hour if not more. Account for that.
	lastRollout := maxt - durToMilis(30*time.Minute)

	// All our series are gauges.
	common := SeriesSpec{
		Targets: 100,
		Type:    Gauge,
		Characteristics: seriesgen.Characteristics{
			Max:            200000000,
			Min:            10000000,
			Jitter:         30000000,
			ScrapeInterval: 15 * time.Second,
			ChangeInterval: 1 * time.Hour,
		},
	}

	// From newest to oldest, in the same way Thanos compactor would do.
	for _, r := range []time.Duration{
		2 * time.Hour,
		2 * time.Hour,
		2 * time.Hour,
		8 * time.Hour,
		8 * time.Hour,
		48 * time.Hour,
		48 * time.Hour,
		48 * time.Hour,
		2 * time.Hour,
	} {
		mint := maxt - durToMilis(r)

		b := BlockSpec{
			Meta: metadata.Meta{
				BlockMeta: tsdb.BlockMeta{
					MaxTime:    maxt,
					MinTime:    mint,
					Compaction: tsdb.BlockMetaCompaction{Level: 1},
					Version:    1,
				},
				Thanos: metadata.Thanos{
					Labels:     extLset.Map(),
					Downsample: metadata.ThanosDownsample{Resolution: 0},
					Source:     "blockgen",
				},
			},
		}

		// Let's say we have 100 applications, 50 metrics each. All rollout every 1h.
		// This makes 2h block to have 10k series, 2d block to have 60k series.

		for {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			smaxt := lastRollout + durToMilis(1*time.Hour)
			if smaxt > maxt {
				smaxt = maxt
			}

			smint := lastRollout
			if smint < mint {
				smint = mint
			}

			for i := 0; i < 50; i++ {
				s := common
				s.Labels = labels.Labels{
					{Name: "__name__", Value: fmt.Sprintf("metric%d", i)},
					{Name: "next_rollout_time", Value: timestamp.Time(lastRollout).String()},
				}
				s.MinTime = smint
				s.MaxTime = smaxt
				b.Series = append(b.Series, s)
			}

			if timestamp.Time(lastRollout).After(timestamp.Time(mint)) {
				break
			}

			lastRollout -= durToMilis(1 * time.Hour)
		}

		if err := blockEncoder(b); err != nil {
			return err
		}
		maxt = mint
	}
	return nil
}

func rangeForTimestamp(t int64, width int64) (maxt int64) {
	return (t/width)*width + width
}

package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"

	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/pkg/labels"
	"github.com/thanos-io/thanos/pkg/store/storepb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerStress(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("stress", "Stress tests a remote StoreAPI.")
	workers := cmd.Flag("workers.num", "Number of go routines for stress testing.").Required().Int()
	timeout := cmd.Flag("timeout", "Timeout of each operation").Default("60s").Duration()
	lookback := cmd.Flag("query.look-back", "How much time into the past at max we should look back").Default("300h").Duration()
	target := cmd.Arg("target", "IP:PORT pair of the target to stress.").TCP()

	// TODO(GiedriusS): send other requests like Info() as well.
	// TODO(GiedriusS): we could ask for random aggregations.
	m["stress"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			conn, err := grpc.Dial((*target).String(), grpc.WithInsecure())
			if err != nil {
				return err
			}
			defer conn.Close()
			c := storepb.NewStoreClient(conn)

			lblvlsCtx, cancel := context.WithTimeout(context.Background(), *timeout)
			defer cancel()

			labelvaluesResp, err := c.LabelValues(lblvlsCtx, &storepb.LabelValuesRequest{Label: labels.MetricName})
			if err != nil {
				return err
			}
			if len(labelvaluesResp.Warnings) > 0 {
				return errors.New(fmt.Sprintf("got %#v warnings from LabelValues() call", labelvaluesResp.Warnings))
			}
			labelvalues := labelvaluesResp.Values
			if len(labelvalues) == 0 {
				return errors.New("the StoreAPI responded with zero metric names")
			}

			g, ctx := errgroup.WithContext(context.Background())

			for i := 0; i < *workers; i++ {
				g.Go(func() error {
					opCtx, cancel := context.WithTimeout(ctx, *timeout)
					defer cancel()

					randomMetric := labelvalues[rand.Intn(len(labelvalues))]
					max := time.Now().Unix()
					min := time.Now().Unix() - rand.Int63n(int64(lookback.Seconds()))

					r, err := c.Series(opCtx, &storepb.SeriesRequest{
						MinTime: min * int64(time.Millisecond),
						MaxTime: max * int64(time.Millisecond),
						Matchers: []storepb.LabelMatcher{
							storepb.LabelMatcher{
								Type:  storepb.LabelMatcher_EQ,
								Name:  labels.MetricName,
								Value: randomMetric,
							},
						},
						Aggregates: []storepb.Aggr{storepb.Aggr_RAW, storepb.Aggr_COUNTER},
					}, grpc.MaxCallRecvMsgSize(math.MaxInt32))

					if err != nil {
						return err
					}

					for {
						seriesResp, err := r.Recv()
						if err == io.EOF {
							break
						}
						if err != nil {
							return err
						}
						series := seriesResp.GetSeries()
						if series == nil {
							continue
						}
					}

					return nil
				})
			}

			return g.Wait()
		}, func(err error) {
			level.Info(logger).Log("msg", "stress test encountered an error", "err", err.Error())
		})
		return nil
	}
}

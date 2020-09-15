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
	"github.com/prometheus/prometheus/pkg/timestamp"
	"github.com/thanos-io/thanos/pkg/store/storepb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	minTime = timestamp.FromTime(time.Unix(math.MinInt64/1000+62135596801, 0))
	maxTime = timestamp.FromTime(time.Unix(math.MaxInt64/1000-62135596801, 999999999))
)

func registerStress(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("stress", "Stress tests a remote StoreAPI.")
	workers := cmd.Flag("workers", "Number of go routines for stress testing.").Required().Int()
	timeout := cmd.Flag("timeout", "Timeout of each operation").Default("60s").Duration()
	lookback := cmd.Flag("query.look-back", "How much time into the past at max we should look back").Default("300h").Duration()
	target := cmd.Arg("target", "IP:PORT pair of the target to stress.").Required().TCP()

	// TODO(GiedriusS): send other requests like Info() as well.
	// TODO(GiedriusS): we could ask for random aggregations.
	m["stress"] = func(g *run.Group, logger log.Logger) error {
		mainCtx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			conn, err := grpc.Dial((*target).String(), grpc.WithInsecure())
			if err != nil {
				return err
			}
			defer conn.Close()
			c := storepb.NewStoreClient(conn)

			lblvlsCtx, lblvlsCancel := context.WithTimeout(mainCtx, *timeout)
			defer lblvlsCancel()

			labelvaluesResp, err := c.LabelValues(lblvlsCtx, &storepb.LabelValuesRequest{
				Label: labels.MetricName,
				Start: minTime,
				End:   maxTime,
			})
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

			errg, errCtx := errgroup.WithContext(mainCtx)

			for i := 0; i < *workers; i++ {
				errg.Go(func() error {
					for {
						select {
						case <-errCtx.Done():
							return nil
						default:
						}

						opCtx, cancel := context.WithTimeout(errCtx, *timeout)
						defer cancel()

						randomMetric := labelvalues[rand.Intn(len(labelvalues))]
						max := time.Now().Unix()
						min := time.Now().Unix() - rand.Int63n(int64(lookback.Seconds()))

						r, err := c.Series(opCtx, &storepb.SeriesRequest{
							MinTime: min * 1000,
							MaxTime: max * 1000,
							Matchers: []storepb.LabelMatcher{
								{
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
							_, err := r.Recv()
							if err == io.EOF {
								break
							}
							if err != nil {
								return err
							}
						}
					}
				})
			}

			return errg.Wait()
		}, func(err error) {
			if err != nil {
				level.Info(logger).Log("msg", "stress test encountered an error", "err", err.Error())
			}
			cancel()
		})
		return nil
	}
}

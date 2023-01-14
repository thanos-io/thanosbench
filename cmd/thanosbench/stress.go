package main

import (
	"context"
	"fmt"
	"io"
	"math"
	"math/rand"
	"time"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/prometheus/prometheus/model/timestamp"
	"github.com/thanos-io/thanos/pkg/store/storepb"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"gopkg.in/alecthomas/kingpin.v2"
)

var (
	minTime = timestamp.FromTime(time.Unix(math.MinInt64/1000+62135596801, 0))
	maxTime = timestamp.FromTime(time.Unix(math.MaxInt64/1000-62135596801, 999999999))
)

func getMetricsFromStore(ctx context.Context, timeout *time.Duration, c storepb.StoreClient) ([]string, error) {
	lblvlsCtx, lblvlsCancel := context.WithTimeout(ctx, *timeout)
	defer lblvlsCancel()

	labelvaluesResp, err := c.LabelValues(lblvlsCtx, &storepb.LabelValuesRequest{
		Label: labels.MetricName,
		Start: minTime,
		End:   maxTime,
	})
	if err != nil {
		return nil, err
	}
	if len(labelvaluesResp.Warnings) > 0 {
		return nil, errors.New(fmt.Sprintf("got %#v warnings from LabelValues() call", labelvaluesResp.Warnings))
	}
	labelvalues := labelvaluesResp.Values
	if len(labelvalues) == 0 {
		return nil, errors.New("the StoreAPI responded with zero metric names")
	}
	return labelvalues, nil
}

func registerStress(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("stress", "Stress tests a remote StoreAPI.")
	workers := cmd.Flag("workers", "Number of go routines for stress testing.").Required().Int()
	timeout := cmd.Flag("timeout", "Timeout of each operation").Default("60s").Duration()
	lookback := cmd.Flag("query.look-back", "How much time into the past at max we should look back").Default("300h").Duration()
	userSpecifiedMetrics := cmd.Flag("metric-name", "Metric to query for").Strings()
	target := cmd.Arg("target", "IP:PORT pair of the target to stress.").Required().TCP()

	// TODO(GiedriusS): send other requests like Info() as well.
	// TODO(GiedriusS): we could ask for random aggregations.
	m["stress"] = func(g *run.Group, logger log.Logger) error {
		mainCtx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			conn, err := grpc.Dial((*target).String(), grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return err
			}
			defer conn.Close()

			c := storepb.NewStoreClient(conn)
			errg, errCtx := errgroup.WithContext(mainCtx)

			var metrics []string
			if *userSpecifiedMetrics != nil && len(*userSpecifiedMetrics) != 0 {
				metrics = *userSpecifiedMetrics
			} else {
				metrics, err = getMetricsFromStore(mainCtx, timeout, c)
				if err != nil {
					return err
				}
			}

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

						randomMetric := metrics[rand.Intn(len(metrics))]
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

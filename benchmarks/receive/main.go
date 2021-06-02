package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/oklog/run"
	"github.com/prometheus/prometheus/prompb"
	"gopkg.in/yaml.v2"
)

type config struct {
	URL     string         `yaml:"url"`
	Tenants []configTenant `yaml:"tenants"`
}

type configTenant struct {
	Name       string             `yaml:"name"`
	Clients    uint               `yaml:"clients"`
	Reqs       uint               `yaml:"reqs"`
	Timeseries []configTimeseries `yaml:"timeseries"`
}

type configTimeseries struct {
	Name   string         `yaml:"name"`
	Labels []configLabels `yaml:"labels"`
}

type configLabels struct {
	Name        string `yaml:"name"`
	Cardinality int    `yaml:"cardinality"`
}

func main() {
	configPath := flag.String("config-file", "./config.yaml", "Path to the config file specifying the way to run the benchmark.")
	flag.Parse()

	configFile, err := os.Open(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	var config config
	if err := yaml.NewDecoder(configFile).Decode(&config); err != nil {
		log.Fatal(err)
	}

	// Generate random strings for label values
	rand.Seed(time.Now().Unix())

	c := Client{
		url:    config.URL,
		client: &http.Client{},
	}

	hostname, err := os.Hostname()
	if err != nil {
		log.Fatal(err)
	}

	var gr run.Group

	for _, tenant := range config.Tenants {
		for i := uint(0); i < tenant.Clients; i++ {
			ctx, cancel := context.WithCancel(context.Background())
			ticker := time.NewTicker(time.Second / time.Duration(tenant.Reqs))
			gr.Add(func() error {

				wreq := generateTimeseries(hostname, tenant.Timeseries)

				for {
					select {
					case <-ctx.Done():
						return nil
					case <-ticker.C:
						if err := c.Request(ctx, tenant.Name, wreq); err != nil {
							log.Println(err)
						}
					}
				}
			}, func(err error) {
				ticker.Stop()
				cancel()
			})
		}

		log.Printf("tenant %s is benchmarking with %dreq/s\n",
			tenant.Name,
			tenant.Clients*tenant.Reqs,
		)
	}

	if err := gr.Run(); err != nil {
		log.Fatal(err)
	}

}

type Client struct {
	client *http.Client
	url    string
}

func (c Client) Request(ctx context.Context, tenant string, wreq prompb.WriteRequest) error {
	nano := time.Now().UnixNano()
	s := []prompb.Sample{{Timestamp: nano / int64(time.Millisecond), Value: float64(nano / int64(time.Second))}}
	for i := range wreq.Timeseries {
		wreq.Timeseries[i].Samples = s
	}

	buf, err := proto.Marshal(&wreq)
	if err != nil {
		return fmt.Errorf("marshalling proto: %w", err)
	}

	req, err := http.NewRequest(http.MethodPost, c.url, bytes.NewBuffer(snappy.Encode(nil, buf)))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req = req.WithContext(ctx)
	req.Header.Add("THANOS-TENANT", tenant)

	resp, err := c.client.Do(req) //nolint:bodyclose
	if err != nil {
		return fmt.Errorf("making request: %w", err)
	}
	defer resp.Body.Close()

	_, err = io.Copy(io.Discard, resp.Body)
	if err != nil {
		return fmt.Errorf("failed to exhaust resp body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("wrong status returned: %s", resp.Status)
	}

	return nil
}

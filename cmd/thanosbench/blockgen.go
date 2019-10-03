package main

import (
	"os"

	"github.com/oklog/run"

	"github.com/thanos-io/thanosbench/pkg/blockgen"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerBlockgen(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("blockgen", "")
	// TODO(bwplotka): Move to pathOrContent from Thanos.
	input := cmd.Flag("input", "Input file for series config.").Required().String()
	outputDir := cmd.Flag("output-dir", "Output directory for generated TSDB data.").Required().String()
	scrapeInterval := cmd.Flag("scrape-interval", "Interval for to generate samples with.").Default("15s").Duration()
	retention := cmd.Flag("retention", "Defines the max time in relation to current time for generated samples.").Required().Duration()

	m["blockgen"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			series, err := blockgen.LoadSeries(*input)
			if err != nil {
				return err
			}

			if err := os.RemoveAll(*outputDir); err != nil {
				level.Error(logger).Log("msg", "remove output dir", "err", err)
				os.Exit(1)
			}

			return blockgen.GenerateTSDB(logger, *outputDir, series, *retention, *scrapeInterval)
		}, func(error) {})
		return nil
	}
}

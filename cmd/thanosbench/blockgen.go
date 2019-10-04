package main

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/thanos-io/thanos/pkg/extflag"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerBlockgen(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("blockgen", "")
	// TODO(bwplotka): Move to pathOrContent from Thanos.
	config := extflag.RegisterPathOrContent(cmd, "config", "YAML file for series config", true)

	outputDir := cmd.Flag("output-dir", "Output directory for generated TSDB data.").Required().String()

	m["blockgen"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			configContent, err := config.Content()
			if err != nil {
				return err
			}
			if err := os.RemoveAll(*outputDir); err != nil {
				return err
			}

			return blockgen.GenerateTSDB(logger, *outputDir, configContent)
		}, func(error) {})
		return nil
	}
}

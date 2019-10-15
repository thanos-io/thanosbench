package main

import (
	"os"

	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/thanos-io/thanos/pkg/extflag"
	"github.com/thanos-io/thanosbench/pkg/walgen"
	"gopkg.in/alecthomas/kingpin.v2"
)

func registerWalgen(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("walgen", "Generates TSDB data into WAL files.")
	config := extflag.RegisterPathOrContent(cmd, "config", "YAML for series config", true)

	outputDir := cmd.Flag("output.dir", "Output directory for generated TSDB data.").Required().String()

	// TODO(bwplotka): Consider mode in which it generates the data only if empty work dir.
	m["walgen"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			configContent, err := config.Content()
			if err != nil {
				return err
			}
			if err := os.RemoveAll(*outputDir); err != nil {
				return err
			}

			return walgen.GenerateTSDB(logger, *outputDir, configContent)
		}, func(error) {})
		return nil
	}
}

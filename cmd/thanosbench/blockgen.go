package main

import (
	"fmt"
	"os"
	"time"

	"github.com/go-kit/kit/log/level"

	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"gopkg.in/alecthomas/kingpin.v2"
)

const (
	defaultProfileName = "zzz"
)

// blockgenProfiles is Hard-coded list of profiles for now.
// Add your own if needed, will make it work nicer later.
var blockgenProfiles = map[string]blockgenProfile{
	defaultProfileName: {
		name:      defaultProfileName,
		outDir:    os.ExpandEnv("${HOME}/zzz-prom-data/zzz"),
		deleteDir: true,
		genConfig: blockgen.GeneratorConfig{
			StartTime:      time.Date(2019, time.September, 30, 0, 0, 0, 0, time.Local),
			SampleInterval: 15 * time.Second,
			FlushInterval:  2 * time.Hour,
			Retention:      10 * time.Hour,
		},
		valConfig: blockgen.ValProviderConfig{
			MetricCount: 200,
			TargetCount: 100,
		},
	},
}

type blockgenProfile struct {
	name      string
	outDir    string
	deleteDir bool
	genConfig blockgen.GeneratorConfig
	valConfig blockgen.ValProviderConfig
}

// Hacky hacky script to generate TSDB
func registerBlockgen(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("blockgen", "Generates Prometheus TSDB blocks.")

	profileName := cmd.Flag("profile.name", "The name of the profile to use.").Required().String()

	m["blockgen"] = func(g *run.Group, logger log.Logger) error {
		g.Add(func() error {
			profile, found := blockgenProfiles[*profileName]
			if !found {
				return fmt.Errorf("profile with name '%s' not found", *profileName)
			}

			if err := execBlockgenProfile(logger, profile); err != nil {
				return errors.Wrap(err, "execBlockgenProfile")
			}

			level.Info(logger).Log("msg", "data generated", "dir", profile.outDir)
			return nil
		}, func(error) {})
		return nil
	}
}

func execBlockgenProfile(logger log.Logger, p blockgenProfile) error {
	level.Info(logger).Log("msg", "running profile", "name", p.name)

	// remove dir if asked to do so
	if p.deleteDir {
		level.Info(logger).Log("msg", "deleting dir", "dir", p.outDir)
		if err := os.RemoveAll(p.outDir); err != nil {
			return errors.Wrapf(err, "delete dir %s", p.outDir)
		}
	}

	writer, err := blockgen.NewBlockWriter(logger, p.outDir)
	if err != nil {
		return errors.Wrap(err, "blockgen.NewBlockWriter")
	}

	valProvider := blockgen.NewValProvider(p.valConfig)
	generator := blockgen.NewGeneratorWithConfig(p.genConfig)

	level.Info(logger).Log("msg", "writing to dir", "dir", p.outDir)
	return generator.Generate(writer, valProvider)
}

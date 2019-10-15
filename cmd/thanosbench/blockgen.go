package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/thanos-io/thanos/pkg/block"

	"github.com/thanos-io/thanos/pkg/extflag"
	"github.com/thanos-io/thanos/pkg/objstore"
	"github.com/thanos-io/thanos/pkg/objstore/client"

	"github.com/go-kit/kit/log/level"

	"github.com/go-kit/kit/log"
	"github.com/oklog/run"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"gopkg.in/alecthomas/kingpin.v2"
)

// registerBlockgen registers blockgen CLI command.
func registerBlockgen(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("blockgen", "Generates Prometheus/Thanos TSDB blocks. Optionally ships blocks to the object storage.")
	config := extflag.RegisterPathOrContent(cmd, "config", "YAML configuration for block generating config.", true)
	outputDir := cmd.Flag("output.dir", "Output directory for generated data.").Required().String()
	objStoreConfig := extflag.RegisterPathOrContent(cmd, "output.objstore.config", "YAML file that contains object store configuration if you want to upload output to the object storage. See format details: https://thanos.io/storage.md/#configuration", false)
	removeAfterUpload := cmd.Flag("output.objstore.remove-local", "If true, blockgen after upload will remove local block").Default("false").Bool()

	m["blockgen"] = func(g *run.Group, logger log.Logger) error {
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			configContent, err := config.Content()
			if err != nil {
				return err
			}

			config, err := blockgen.LoadConfig(configContent)
			if err != nil {
				return err
			}

			level.Info(logger).Log("msg", "loaded config", "config", fmt.Sprintf("%#v", config))
			objstoreConfig, err := objStoreConfig.Content()
			if err != nil {
				return err
			}

			var syncer blockgen.Syncer = blockgen.NoopSyncer{}
			if len(objstoreConfig) > 0 {
				bkt, err := client.NewBucket(logger, objstoreConfig, nil, "blockgen")
				if err != nil {
					return err
				}
				level.Info(logger).Log("msg", "uploading enabled", "bucket", bkt.Name())
				syncer = &bucketSyncer{logger: logger, bkt: bkt, remove: *removeAfterUpload}
			} else {
				level.Info(logger).Log("msg", "no upload config found; uploading disabled")
			}

			return blockgen.Generate(ctx, logger, *outputDir, syncer, config)
		}, func(error) { cancel() })
		return nil
	}
}

type bucketSyncer struct {
	logger log.Logger
	bkt    objstore.Bucket
	remove bool
}

func (s *bucketSyncer) Sync(ctx context.Context, bdir string) error {
	start := time.Now()
	if err := block.Upload(ctx, s.logger, s.bkt, bdir); err != nil {
		return err
	}
	level.Info(s.logger).Log("msg", "uploaded block to bucket", "bdir", bdir, "elapsed", time.Since(start))
	if s.remove {
		if err := os.RemoveAll(bdir); err != nil {
			return err
		}
		level.Info(s.logger).Log("msg", "removed local block", "bdir", bdir)
	}
	return nil
}

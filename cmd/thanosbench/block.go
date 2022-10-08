package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"

	extflag "github.com/efficientgo/tools/extkingpin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/pkg/errors"
	promModel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/labels"
	"github.com/thanos-io/objstore"
	"github.com/thanos-io/objstore/client"
	"github.com/thanos-io/thanos/pkg/block"
	"github.com/thanos-io/thanos/pkg/block/metadata"
	"github.com/thanos-io/thanos/pkg/extkingpin"
	"github.com/thanos-io/thanos/pkg/model"
	"github.com/thanos-io/thanosbench/pkg/blockgen"
	"gopkg.in/alecthomas/kingpin.v2"
	"gopkg.in/yaml.v2"
)

func millisToDur(t int64) time.Duration {
	return time.Duration(t * int64(time.Millisecond))
}

func printBlocks(bts ...blockgen.BlockSpec) string {
	var msg []string
	for _, b := range bts {
		msg = append(msg, fmt.Sprintf("[%d - %d](%s) ", b.MinTime, b.MaxTime, millisToDur(b.MaxTime-b.MinTime).String()))
	}
	return strings.Join(msg, ",")
}

func registerBlock(m map[string]setupFunc, app *kingpin.Application) {
	cmd := app.Command("block", "Tools for generating TSDB/Prometheus blocks")
	registerBlockGen(m, cmd)
	registerBlockPlan(m, cmd)
}
func registerBlockGen(m map[string]setupFunc, root *kingpin.CmdClause) {
	cmd := root.Command("gen", "Generates Prometheus/Thanos TSDB blocks from input. Expects []blockgen.BlockSpec in YAML format as input.")
	config := extflag.RegisterPathOrContent(cmd, "config", "YAML for  []blockgen.BlockSpec. Leave this empty in order to be able to pass this through STDIN", extflag.WithEnvSubstitution())
	objStore := *extkingpin.RegisterCommonObjStoreFlags(cmd, "", false)
	outputDir := cmd.Flag("output.dir", "Output directory for generated data.").Required().String()
	workers := cmd.Flag("workers", "Number of go routines for block generation. If 0, 2*runtime.GOMAXPROCS(0) is used.").Int()
	m["block gen"] = func(g *run.Group, logger log.Logger) error {
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			goroutines := *workers
			if goroutines == 0 {
				goroutines = 2 * runtime.GOMAXPROCS(0)
			}

			cfg, err := config.Content()
			if err != nil {
				return err
			}

			objStoreContentYaml, err := objStore.Content()
			if err != nil {
				return errors.Wrap(err, "getting object store config")
			}

			var (
				upload bool
				bkt    objstore.InstrumentedBucket
			)
			if len(objStoreContentYaml) == 0 {
				level.Info(logger).Log("msg", "no supported bucket was configured, uploads will be disabled")
			} else {
				upload = true
				bkt, err = client.NewBucket(logger, objStoreContentYaml, nil, "blockgen")
				if err != nil {
					return err
				}
			}

			n := 0
			if len(cfg) > 0 {
				bs := []blockgen.BlockSpec{}
				if err := yaml.UnmarshalStrict(cfg, &bs); err != nil {
					return err
				}
				for _, b := range bs {
					level.Info(logger).Log("msg", "generating block", "spec", printBlocks(b))
					id, err := blockgen.Generate(ctx, logger, goroutines, *outputDir, b)
					if err != nil {
						return errors.Wrap(err, "generate")
					}
					n++
					blockDir := path.Join(*outputDir, id.String())
					level.Info(logger).Log("msg", "generated block", "path", blockDir, "count", n)
					runtime.GC()

					if upload {
						if err := block.Upload(ctx, logger, bkt, blockDir, metadata.NoneFunc); err != nil {
							return errors.Wrapf(err, "upload block %s", id)
						}
						level.Info(logger).Log("msg", "uploaded block to object storage", "path", blockDir)
					}
				}
				return ctx.Err()
			}

			dec := yaml.NewDecoder(os.Stdin)
			dec.SetStrict(true)
			for ctx.Err() == nil {
				b := blockgen.BlockSpec{}
				err := dec.Decode(&b)
				if err == io.EOF {
					level.Info(logger).Log("msg", "all blocks done", "count", n)
					return nil
				}
				if err != nil {
					return errors.Wrap(err, "decode")
				}

				level.Info(logger).Log("msg", "generating block", "spec", printBlocks(b))
				id, err := blockgen.Generate(ctx, logger, goroutines, *outputDir, b)
				if err != nil {
					return errors.Wrap(err, "generate")
				}
				n++
				blockDir := path.Join(*outputDir, id.String())
				level.Info(logger).Log("msg", "generated block", "path", blockDir, "count", n)
				runtime.GC()

				if upload {
					if err := block.Upload(ctx, logger, bkt, blockDir, metadata.NoneFunc); err != nil {
						return errors.Wrapf(err, "upload block %s", id)
					}
					level.Info(logger).Log("msg", "uploaded block to object storage", "path", blockDir)
				}
			}
			return ctx.Err()
		}, func(error) { cancel() })
		return nil
	}
}

func parseFlagLabels(s []string) (labels.Labels, error) {
	var lset labels.Labels
	for _, l := range s {
		parts := strings.SplitN(l, "=", 2)
		if len(parts) != 2 {
			return nil, errors.Errorf("unrecognized label %q", l)
		}
		if !promModel.LabelName.IsValid(promModel.LabelName(parts[0])) {
			return nil, errors.Errorf("unsupported format for label %s", l)
		}
		val, err := strconv.Unquote(parts[1])
		if err != nil {
			return nil, errors.Wrap(err, "unquote label value")
		}
		lset = append(lset, labels.Label{Name: parts[0], Value: val})
	}
	return lset, nil
}

func registerBlockPlan(m map[string]setupFunc, root *kingpin.CmdClause) {
	cmd := root.Command("plan", `Plan generates blocks specs used by blockgen command to build blocks. 

Example plan with generation:

./thanosbench block plan -p <profile> --labels 'cluster="one"' --max-time 2019-10-18T00:00:00Z | ./thanosbench block gen --output.dir ./genblocks --workers 20`)
	profile := cmd.Flag("profile", "Name of the harcoded profile to use").Required().Short('p').Enum(blockgen.Profiles.Keys()...)
	maxTime := model.TimeOrDuration(cmd.Flag("max-time", "If empty current time - 30m (usual consistency delay) is used.").Default("30m"))
	extLset := cmd.Flag("labels", "External labels for block stream (repeated).").PlaceHolder("<name>=\"<value>\"").Strings()
	m["block plan"] = func(g *run.Group, _ log.Logger) error {
		ctx, cancel := context.WithCancel(context.Background())
		g.Add(func() error {
			lset, err := parseFlagLabels(*extLset)
			if err != nil {
				return err
			}
			planFn := blockgen.Profiles[*profile]

			enc := yaml.NewEncoder(os.Stdout)
			return planFn(ctx, *maxTime, lset, func(spec blockgen.BlockSpec) error { return enc.Encode(spec) })
		}, func(error) { cancel() })
		return nil
	}
}

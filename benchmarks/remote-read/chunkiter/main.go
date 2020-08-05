package main

import (
	"github.com/bwplotka/mimic"
	"github.com/thanos-io/thanosbench/benchmarks"
	"github.com/thanos-io/thanosbench/configs/abstractions/dockerimage"
	k8s "github.com/thanos-io/thanosbench/configs/kubernetes"
	"gopkg.in/alecthomas/kingpin.v2"
)

func main() {
	generator := mimic.New(func(cmd *kingpin.CmdClause) {
		cmd.GetFlag("output").Default("benchmarks/remote-read/chunkiter/manifests")
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	k8s.GenRemoteReadBenchPrometheusWith1MoBlocks1kSeries(
		generator,
		"prometheus",
		benchmarks.Namespace,
		dockerimage.PublicPrometheus("v2.20.0"),
		dockerimage.PublicThanos("v0.14.0"),
	)
	k8s.GenRemoteReadBenchPrometheusWith1MoBlocks1kSeries(
		generator,
		"prometheus-chunkiter",
		benchmarks.Namespace,
		dockerimage.Image{Organization: "quay.io/thanos", Project: "prometheus", Version: "pre-2.21-118aeab02"},
		dockerimage.PublicThanos("v0.14.0"),
	)
}

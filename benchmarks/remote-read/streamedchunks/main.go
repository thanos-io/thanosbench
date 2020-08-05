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
		cmd.GetFlag("output").Default("benchmarks/remote-read/streamedchunks/manifests")
	})

	// Make sure to generate at the very end.
	defer generator.Generate()

	k8s.GenRemoteReadBenchPrometheusWith10h10kSeriesWAL(generator, "prometheus", benchmarks.Namespace, dockerimage.PublicPrometheus("v2.12.0"), dockerimage.PublicThanos("v0.7.0"))
	k8s.GenRemoteReadBenchPrometheusWith10h10kSeriesWAL(generator, "prometheus-rr-streamed", benchmarks.Namespace, dockerimage.PublicPrometheus("v2.13.0"), dockerimage.PublicThanos("v0.7.0"))
}
